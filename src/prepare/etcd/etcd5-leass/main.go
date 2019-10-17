package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

func main() {
	var(
		config clientv3.Config
		client *clientv3.Client
		err error

		kv clientv3.KV
		putResp *clientv3.PutResponse
		getResp *clientv3.GetResponse

		// lease租约
		lease clientv3.Lease
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		// 租约续租响应
		leaseKeepAliveResp *clientv3.LeaseKeepAliveResponse
		// 只读channel
		leaseKeepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
	)
	config = clientv3.Config{
		Endpoints:[]string{"192.168.174.134:2379"},
		DialTimeout:5*time.Second,
	}
	if client,err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	// 创建lease对象 (房东)
	lease = clientv3.NewLease(client)
	// 申请1个10秒的lease租约 (类似于和房东续租)
	// Grant ttl 过期时间 单位秒
	if leaseGrantResp,err = lease.Grant(context.TODO(),10); err != nil{
		panic(err)
	}
	// 获取 租约id
	leaseId = leaseGrantResp.ID

	// lease.KeepAlive() 高级API
	// etcd客户端启动1个协程,在协程里自动续租,发心跳到etcd
	// lease.KeepAliveOnce() 低级API
	// 只续租1次 需要自己写for循环去续租

	// 自动续租
	// 返回channel 每一次发送出去的续租请求的应答 它自动续租请求response也会不断返回

	// TODO NOTICE 演示主动取消context 租约是10秒 5秒后取消主动续租 = 10+5=15秒
	// 5秒后 该ctx超时结束被取消掉 KeepAlive 续租协程会被终止 会向 leaseKeepAliveRespChan 投递nil应答 goto END
	ctx,_ := context.WithTimeout(context.TODO(),5 * time.Second)
	if leaseKeepAliveRespChan,err = lease.KeepAlive(ctx,leaseId); err != nil {
		panic(err)
	}

	// TODO:NOTICE 如果不使用超时 context 主动取消续租 在客户端与etcd服务端网络连通情况下 续租永远成功 key 将永远存在不会被删除
//	if leaseKeepAliveRespChan,err = lease.KeepAlive(context.TODO(),leaseId); err != nil {
//		panic(err)
//	}

	// etcd客户端启动1个协程,在协程里自动续租,发心跳到etcd
	// 开启1个协程去消费 处理续租应答
	go func() {
		for{
			select {
			// 取出每次心跳的应答
			case leaseKeepAliveResp = <- leaseKeepAliveRespChan:
				// 续租过程中出现异常 etcd客户端和服务端失联很久,后来联通,发现租约已经过期
				// leaseKeepAliveRespChan 就会投递1个 leaseKeepAliveResp==nil 的应答
				if leaseKeepAliveResp == nil {
					fmt.Println("因为异常(网络异常/context主动取消)etcd客户端如法向服务端报送心跳,租约失效")
					// 结束该协程
					goto END
				} else {
					// 大概每秒会续租1次 每秒收到1次etcd服务端返回的应答
					fmt.Println("续租成功:",leaseKeepAliveResp.ID)
				}
			}
		}
		END:
	}()

	// put kv 与 租约关联 如果不续租10秒自动过期删除
	kv = clientv3.NewKV(client)
	if putResp,err = kv.Put(context.TODO(),"/cron/lock/job1","",clientv3.WithLease(leaseId)); err != nil {
		panic(err)
	}
	fmt.Printf("写入成功 Revision = %v: now = %v \n",putResp.Header.Revision,time.Now())

	// 定时看key是否过期
	for {
		if getResp,err = kv.Get(context.TODO(),"/cron/lock/job1"); err != nil {
			panic(err)
		}
		if getResp.Count == 0{
			fmt.Printf("kv过期 now = %v \n",time.Now())
			break
		}
		fmt.Println("kv没过期:",getResp.Kvs)
		time.Sleep(1 * time.Second)
	}
}

/*
取消租约续租演示:

$ go run src/prepare/etcd/etcd5-leass/main.go
续租成功: 7587841726481637888
写入成功 Revision = 14: now = 2019-10-17 21:31:38.6039052 +0800 CST m=+0.089088001
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
续租成功: 7587841726481637888
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
因为异常(网络异常/context主动取消)etcd客户端如法向服务端报送心跳,租约失效
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv没过期: [key:"/cron/lock/job1" create_revision:14 mod_revision:14 version:1 lease:7587841726481637888 ]
kv过期 now = 2019-10-17 21:31:52.6738454 +0800 CST m=+14.159028201

52 - 38 + 1 = 15秒
*/

/*
[root@CentOS3 ~]# etcdctl watch "/cron/lock" --prefix
PUT
/cron/lock/job1

DELETE
/cron/lock/job1

*/