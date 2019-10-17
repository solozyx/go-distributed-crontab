package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

func main(){
	var(
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		getResp *clientv3.GetResponse
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
	// context.TO_DO() 不需要控制超时时间不需要主动取消请求 todo即可
	// WithPrevKv()在get用不到
	if getResp,err = kv.Get(context.TODO(),"/cron/jobs/job7"); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(getResp.Kvs)
	}

	if getResp,err = kv.Get(context.TODO(),"/cron/jobs/job7",clientv3.WithCountOnly()); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(getResp.Count)
	}

	fmt.Println("----------------------")

	// 读取以 "/cron/jobs/" 为前缀的所有key
	// 在jobs后的/不能省略确保是读取该目录下的内容
	if getResp,err = kv.Get(context.TODO(),"/cron/jobs/",clientv3.WithPrefix()); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(getResp.Kvs,getResp.Count)
	}
}

/*
$ go run etcd3-getkv/main.go
[key:"/cron/jobs/job7" create_revision:3 mod_revision:5 version:3 value:"{\"jobName\":\"job7-prevkv-test-4\"}" ]
1
----------------------
[key:"/cron/jobs/job7" create_revision:3 mod_revision:5 version:3 value:"{\"jobName\":\"job7-prevkv-test-4\"}" ] 1
*/