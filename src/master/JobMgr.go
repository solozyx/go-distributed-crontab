package master

import (
	"common"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var (
	// 单例
	G_jobMgr *JobMgr
)

/*
任务管理器
*/
type JobMgr struct {
	// etcd客户端指针
	client *clientv3.Client
	// KV存储接口
	kv clientv3.KV
	// 租约接口
	lease clientv3.Lease
}

/*
初始化任务管理器 建立和etcd的连接
*/
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)
	// 初始化配置
	config = clientv3.Config{
		// etcd 集群地址
		// Endpoints:[]string{""},
		Endpoints: G_config.EtcdEndpoints,
		// 超时时间
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	// 客户端连接到etcd服务器
	if client, err = clientv3.New(config); err != nil {
		return
	}
	// KV lease
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	// 赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

/*
保存job到etcd服务器
保存成功,返回旧的job,新键 / 旧键更新覆盖旧值 把旧值返回
保存etcd格式 :
	/cron/jobs/jobName -> json
*/
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)
	// etcd服务端保存key value
	jobKey = common.JOB_SAVE_DIR + job.Name
	// json.Marshal(job) 传入结构体本身 / 结构体指针 都可以
	if jobValue, err = json.Marshal(job); err != nil {
		// 有错误会赋值给返回参数列表的err return回去
		return
	}
	// 保存到etcd
	// []byte <-> string 转换是无缝的,底层存储类型都是byte
	if putResp, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		fmt.Println("etcd 保存 job失败")
		return
	}
	// 更新则返回旧值
	if putResp.PrevKv != nil {
		// 旧值 -> 反序列化 Job
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			// 旧值非法并不会影响新值的设置,这里put 操作已经成功
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

/*
删除任务
删除成功 返回被删除的任务,删除不存在的任务返回nil空指针
		 etcd删除不存在的key不会报错
*/
func (jobMgr *JobMgr) DeleteJob(jobName string) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)
	// etcd保存任务key
	jobKey = common.JOB_SAVE_DIR + jobName
	// etcd删除任务
	if delResp, err = jobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		// 删除失败 告诉调用者
		return
	}
	// 返回被删除任务 1真实删除
	// NOTICE 不可delResp.PrevKvs[0]取不到 0要删除的key不存在
	if len(delResp.PrevKvs) > 0 {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			// json -> Job 失败 不关心旧值 只要删除了就是成功 err = nil
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

/*
不支持翻页 etcd get 某个目录下的所有kv
*/
func (jobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		// 指针变量用于Job的反序列化
		job *common.Job
	)
	dirKey = common.JOB_SAVE_DIR
	if getResp, err = jobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		// err 返回给调用者
		return
	}
	// 返回参数列表 (jobList []*common.Job,err error) 声明了一个slice 是nil空指针
	// 需要make空间初始化 长度为0的slice不是nil
	// NOTICE 后续调用者只需要判断 jobList的长度是否0 不需要判断它是否nil空指针 简化调用者判断复杂性
	// len(jobList) == 0
	jobList = make([]*common.Job, 0)

	// 遍历所有kv json -> 反序列化 Job
	for _, kvPair = range getResp.Kvs {
		// 给指针变量job 赋值新的 &Job{} 对象 去做反序列化
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			// 忽略 json字符串 -> Job对象 的错误
			err = nil
			continue
		}
		jobList = append(jobList, job)
	}
	return
}

/*
杀死任务
更新 key = /cron/killer/jobName 所有的worker监听者watch到 key-value 变化 杀死任务
*/
func (jobMgr *JobMgr) KillJob(jobName string) (err error) {
	var (
		killerKey      string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)
	// NOTICE 通知worker节点杀死对应任务
	killerKey = common.JOB_KILL_DIR + jobName
	// 让worker监听到一次put操作,worker杀死任务
	// 设置租约 设置1个过期时间 /cron/killer/ 目录下的存储会过期掉
	// 不要浪费etcd的存储空间
	// 带着租约去put一次 put完成马上过期掉
	if leaseGrantResp, err = jobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	// 租约id
	leaseId = leaseGrantResp.ID
	// 设置killer标记 空值即可
	// 1秒后自动删除 worker节点监听put操作 所以不影响监听
	// 不关心put的结果 只关心是否put成功
	if _, err = jobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}
	return
}
