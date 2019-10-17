package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

func main() {
	var(
		// etcd 客户端配置
		config clientv3.Config
		// etcd 客户端
		client *clientv3.Client
		err error
		// KV
		kv clientv3.KV
		putResp *clientv3.PutResponse
	)

	config = clientv3.Config{
		// 集群列表
		Endpoints:[]string{"192.168.174.134:2379"},
		// 超时时间
		DialTimeout:5*time.Second,
	}
	// 创建 etcd 客户端
	if client,err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	// 读写 etcd KV 存储
	kv = clientv3.NewKV(client)

	// context.Context 用于取消本次Put操作,网络交互可能超时
	// context.TO_DO() 代表什么也不做 该ctx用于不会被取消 占位
	// val := "{\"jobName\":\"job7-prevkv-test-2\"}"
	// val := "{\"jobName\":\"job7-prevkv-test-3\"}"
	val := "{\"jobName\":\"job7-prevkv-test-4\"}"
	if putResp,err = kv.Put(context.TODO(),"/cron/jobs/job7",val,clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("/cron/jobs/job7 - Revision : ",putResp.Header.Revision)
		// 如果put一个新key PrevKv 是空指针
		if putResp.PrevKv != nil {
			fmt.Println("/cron/jobs/job7 - PrevKey   = ",string(putResp.PrevKv.Key))
			fmt.Println("/cron/jobs/job7 - PrevValue = ",string(putResp.PrevKv.Value))
		}
	}
}

/*
[root@CentOS3 ~]# etcdctl watch "/cron/jobs" --prefix
PUT
/cron/jobs/job7
{"jobName":"job7-prevkv-test-2"}
PUT
/cron/jobs/job7
{"jobName":"job7-prevkv-test-3"}
PUT
/cron/jobs/job7
{"jobName":"job7-prevkv-test-4"}

-----------------------------------------------------------------

$ go run etcd2-putkv/main.go
/cron/jobs/job7 - Revision :  3

$ go run etcd2-putkv/main.go
/cron/jobs/job7 - Revision :  4
/cron/jobs/job7 - PrevKey   =  /cron/jobs/job7
/cron/jobs/job7 - PrevValue =  {"jobName":"job7-prevkv-test-2"}

$ go run etcd2-putkv/main.go
/cron/jobs/job7 - Revision :  5
/cron/jobs/job7 - PrevKey   =  /cron/jobs/job7
/cron/jobs/job7 - PrevValue =  {"jobName":"job7-prevkv-test-3"}

*/