package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

func main()  {
	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		// 接口类型
		kv clientv3.KV
		getResp *clientv3.GetResponse
		watchStartRevision int64
		watcher clientv3.Watcher
		watchResp clientv3.WatchResponse
		watchRespChan <- chan clientv3.WatchResponse
		event *clientv3.Event
	)
	config = clientv3.Config{
		Endpoints:[]string{"192.168.174.134:2379"},
		DialTimeout:5*time.Second,
	}
	if client,err = clientv3.New(config); err != nil {
		panic(err)
	}

	kv = clientv3.NewKV(client)
	// 开启1个协程模拟etcd中KV的变化
	go func() {
		for {
			kv.Put(context.TODO(),"/cron/jobs/job1","{\"jobName\":\"job1\"}")
			kv.Delete(context.TODO(),"/cron/jobs/job1")
			time.Sleep(1 * time.Second)
		}
	}()

	// 先GET到当前值 并监听后续变化
	if getResp,err = kv.Get(context.TODO(),"/cron/jobs/job1"); err != nil {
		panic(err)
	}
	if len(getResp.Kvs) > 0 {
		fmt.Println("当前值 : ",getResp.Kvs[0].Key,getResp.Kvs[0].Value)
	}

	// getResp.Header.Revision 当前etcd集群事务ID 单调递增 put/del 都会导致 当前 Revision+1
	// watch "/cron/jobs/job1"  该key的事务
	// 设置监听开始 事务ID
	watchStartRevision = getResp.Header.Revision + 1
	watcher = clientv3.NewWatcher(client)

	// 启动监听
	fmt.Printf("从 %d 事务id开始向后监听\n",watchStartRevision)
	// 监听到key发生变化发到channel
	// TODO 通过ctx取消监听 ctx取消 watchRespChan 被close掉 下面的for range就会退出
	ctx,cancelFunc := context.WithCancel(context.TODO())
	time.AfterFunc(5 * time.Second, func() {
		cancelFunc()
	})
	watchRespChan = watcher.Watch(ctx,"/cron/jobs/job1",clientv3.WithRev(watchStartRevision))

	// watchRespChan = watcher.Watch(context.TO-DO(),"/cron/jobs/job1",clientv3.WithRev(watchStartRevision))
	// 轮询 watchRespChan 如果kv发生变化会投递到channel
	// 用 for 循环 和管道读出符号 <- channel 取
	// 也可以 for range 遍历
	for watchResp = range watchRespChan {
		for _,event = range  watchResp.Events {
			switch event.Type {
			case mvccpb.PUT:
				fmt.Println("key : ",string(event.Kv.Key),"修改为 : ",string(event.Kv.Value),
					"CreateRevision : ",event.Kv.CreateRevision,"ModifyRevision : ",event.Kv.ModRevision)
			case mvccpb.DELETE:
				fmt.Println("key : ",string(event.Kv.Key),
					"删除了 ModRevision : ",event.Kv.ModRevision)
			}
		}
	}
}

/*
[root@CentOS3 ~]# etcdctl watch "/cron/jobs" --prefix
PUT
/cron/jobs/job1
{"jobName":"job1"}
DELETE
/cron/jobs/job1

PUT
/cron/jobs/job1
{"jobName":"job1"}
DELETE
/cron/jobs/job1

PUT
/cron/jobs/job1
{"jobName":"job1"}
DELETE
/cron/jobs/job1

PUT
/cron/jobs/job1
{"jobName":"job1"}
DELETE
/cron/jobs/job1

PUT
/cron/jobs/job1
{"jobName":"job1"}
DELETE
/cron/jobs/job1

*/

/*
$ go run etcd6-watch/main.go
从 18 事务id开始向后监听
key :  /cron/jobs/job1 修改为 :  {"jobName":"job1"} CreateRevision :  18 ModifyRevision :  18
key :  /cron/jobs/job1 删除了 ModRevision :  19
key :  /cron/jobs/job1 修改为 :  {"jobName":"job1"} CreateRevision :  20 ModifyRevision :  20
key :  /cron/jobs/job1 删除了 ModRevision :  21
key :  /cron/jobs/job1 修改为 :  {"jobName":"job1"} CreateRevision :  22 ModifyRevision :  22
key :  /cron/jobs/job1 删除了 ModRevision :  23
key :  /cron/jobs/job1 修改为 :  {"jobName":"job1"} CreateRevision :  24 ModifyRevision :  24
key :  /cron/jobs/job1 删除了 ModRevision :  25
key :  /cron/jobs/job1 修改为 :  {"jobName":"job1"} CreateRevision :  26 ModifyRevision :  26
key :  /cron/jobs/job1 删除了 ModRevision :  27

*/