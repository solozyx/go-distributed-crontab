package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

func main() {
	var(
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		deleteResp *clientv3.DeleteResponse
		index int
		kvPair *mvccpb.KeyValue
	)
	config = clientv3.Config{
		Endpoints:[]string{"192.168.174.134:2379"},
		DialTimeout:5*time.Second,
	}

	if client,err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	kv = clientv3.NewKV(client)
	// context.TO_DO() golang几乎所有API都支持ctx操作 用来取消调用
	// 调用可能是走网络的 代码逻辑不想等太久 通过ctx做取消
	// TO-DO 不做定制化超时/取消控制
	// key 删除某个key
	// opts
	if deleteResp,err = kv.Delete(context.TODO(),"/cron/jobs/job1",clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
		return
	}
	// 被删除之前kv
	// 删除请求key可能不存在 请求成功 可能删除的kv是0 Deleted==0 PrevKvs是空
	if len(deleteResp.PrevKvs) > 0 {
		for index,kvPair = range deleteResp.PrevKvs {
			fmt.Println("删除了:",index,string(kvPair.Key),string(kvPair.Value))
		}
	}

	fmt.Println("-------------")

	// deleteResp,err = kv.Delete(context.TODO(),"/cron/jobs/",clientv3.WithPrefix())

	// deleteResp,err = kv.Delete(context.TODO(),"/cron/jobs/job7",clientv3.WithFromKey(),clientv3.WithLimit(2))
	deleteResp,err = kv.Delete(context.TODO(),"/cron/jobs/job7",clientv3.WithFromKey(),clientv3.WithLimit(1))
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(deleteResp.PrevKvs) > 0 {
		for index,kvPair = range deleteResp.PrevKvs {
			fmt.Println("删除了:",index,string(kvPair.Key),string(kvPair.Value))
		}
	}
}

/*
[root@CentOS3 ~]# etcdctl get "/cron/jobs" --prefix
/cron/jobs/job1
{"jobName":job1}
/cron/jobs/job7
{"jobName":"job7-prevkv-test-4"}
[root@CentOS3 ~]#

-----------------------------------------------------

$ go run etcd4-deletekv/main.go
删除了: 0 /cron/jobs/job1 {"jobName":job1}
-------------
panic: unexpected limit in delete

[root@CentOS3 ~]# etcdctl watch "/cron/jobs" --prefix
DELETE
/cron/jobs/job1
*/