package worker

import (
	"github.com/coreos/etcd/clientv3"
	"time"
	"context"
	"common"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"fmt"
)

var(
	G_jobMgr *JobMgr
)

/*
worker JobMgr 任务管理器 监听 /cron/jobs/ 任务变化
master增删改查修改etcd中的任务
worker监听etcd中的任务同步到内存 监听kv变化
*/
type JobMgr struct{
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
	// etcd集群监听器
	watcher clientv3.Watcher
}

func InitJobMgr() (err error) {
	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
		// etcd集群监听器
		watcher clientv3.Watcher
	)
	config = clientv3.Config{
		Endpoints:G_config.EtcdEndpoints,
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	if client,err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	G_jobMgr = &JobMgr{
		client:client,
		kv:kv,
		lease:lease,
		watcher:watcher,
	}
	//启动任务监听
	G_jobMgr.watchJobs()
	//启动强杀任务监听
	G_jobMgr.watchKiller()
	return
}

/*
私有方法
监听etcd中 /cron/jobs/ 和 /cron/killer/ 任务变化
*/
func (jobMgr *JobMgr)watchJobs()(err error){
	var(
		jobName string
		job *common.Job
		jobEvent *common.JobEvent
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		watchStartRevision int64
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		watchEvent *clientv3.Event
	)
	// 1.get /cron/jobs/ 目录下所有任务 获取当前etcd集群Revision
	if getResp,err = jobMgr.kv.Get(context.TODO(),common.JOB_SAVE_DIR,clientv3.WithPrefix()); err != nil{
		// 从etcd集群get失败 return err 给调用者
		return
	}
	// 启动后 获取当前全量任务列表 把存量全量任务同步给 worker 的 Scheduler
	// 遍历当前etcd中任务 kvPair.Value json字符串 ->反序列为 Job 对象
	for _,kvPair = range getResp.Kvs {
		if job,err = common.UnpackJob(kvPair.Value); err == nil { // err!=nil 任务反序列化失败 静默处理
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE,job)
			// TODO 把 jobEvent 推送给 scheduler 调度协程调度任务
			G_scheduler.PushJobEvent(jobEvent)
		}
	}
	// 2.从该Revision开始向后监听kv变化事件 启动监听协程
	go func(){
		// 对etcd的修改导致集群Revision+1 从这次修改开始监听
		watchStartRevision = getResp.Header.Revision + 1
		// 监听etcd集群变化 /cron/jobs/ 后续变化
		// NOTICE channel在etcd设计中的表现
		watchChan = jobMgr.watcher.Watch(
			context.TODO(),
			common.JOB_SAVE_DIR,
			clientv3.WithRev(watchStartRevision),clientv3.WithPrefix())
		for watchResp = range watchChan {
			// 每次etcd监听返回的是多个Event事件
			// 每个Event是不同key的事件 不一定是同一个key
			// Event有类型
			for _,watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 新建 修改
				// job json string byte -> 反序列化 *Job 推送一个更新事件给 scheduler调度协程
				if job,err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
					// json字符串反序列化失败 非法json 一般不会出现 静默处理 继续监听后续变化
					continue
				}
				// 构建一个更新Event事件
				jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE,job)
					fmt.Println(jobEvent)
				case mvccpb.DELETE: // 删除
				// 删除了 /cron/jobs/job1 需要得到 job1
				jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
				// 删除job只需要jobName 其他字段可忽略
				job = &common.Job{Name:jobName}
				// 构造一个删除Event
				jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE,job)
					fmt.Println(jobEvent)
				// 推送一个删除事件给scheduler
				}

				// TODO 把 jobEvent 推送给 scheduler 调度协程调度任务
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()
	return
}

/*
TODO 在分布式集群中防止1个任务并发调度多次 在分布式环境下做互斥【分布式锁】
锁到了执行下面的代码 没锁到就跳过了
因为其他worker执行了 本worker节点就不需要执行了

锁 jobName 任务 仅仅是把锁创建出来 并不加锁
*/
func (jobMgr *JobMgr)CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobName,jobMgr.kv,jobMgr.lease)
	return
}

/*
私有方法
监听etcd中 /cron/killer/ 任务变化
*/
func (jobMgr *JobMgr)watchKiller()(err error){
	var(
		jobName string
		job *common.Job
		jobEvent *common.JobEvent
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		watchEvent *clientv3.Event
	)
	go func(){
		watchChan = jobMgr.watcher.Watch(context.TODO(),common.JOB_KILL_DIR,clientv3.WithPrefix())
		for watchResp = range watchChan {
			for _,watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				// 强杀某个任务
				case mvccpb.PUT:
					// /cron/killer/job1 提前 job1
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					fmt.Println("worker JobMgr 推送强杀任务 : ",jobName)
					// 强杀任务只需要任务名
					job = &common.Job{
						Name:jobName,
					}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL,job)
					// kill 事件推送给 Scheduler 调度模块
					G_scheduler.PushJobEvent(jobEvent)
				// killer标记过期被自动删除
				case mvccpb.DELETE:
					// delete操作不关心 租约过期 key 被删除的情形不关心
				}
			}
		}
	}()
	return
}