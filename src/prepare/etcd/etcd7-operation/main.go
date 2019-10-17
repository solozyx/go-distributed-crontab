package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

func main() {
	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		putOp clientv3.Op
		getOp clientv3.Op
		opResp clientv3.OpResponse
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

	// 创建Op operation
	putOp = clientv3.OpPut("/cron/jobs/job1","job1")
	// 执行Op
	if opResp,err = kv.Do(context.TODO(),putOp); err != nil {
		panic(err)
	}
	fmt.Println("Put Revision : ",opResp.Put().Header.Revision)

	getOp = clientv3.OpGet("/cron/jobs/job1")
	if opResp,err = kv.Do(context.TODO(),getOp); err != nil {
		panic(err)
	}
	fmt.Println("Get Revision : ",opResp.Get().Header.Revision,
		string(opResp.Get().Kvs[0].Key),string(opResp.Get().Kvs[0].Key))
}

/*
$ go run src/prepare/etcd/etcd7-operation/main.go
Put Revision :  28
Get Revision :  28 /cron/jobs/job1 /cron/jobs/job1

[root@CentOS3 ~]# etcdctl watch "/cron/jobs" --prefix
PUT
/cron/jobs/job1
job1

*/