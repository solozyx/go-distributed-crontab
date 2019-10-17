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
		// 租约
		lease clientv3.Lease
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		leaseKeepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeepAliveResp *clientv3.LeaseKeepAliveResponse
		// 取消自动续租
		ctx context.Context
		cancelFunc context.CancelFunc
		// 事务
		txn clientv3.Txn
		txnResp *clientv3.TxnResponse
	)
	config = clientv3.Config{
		Endpoints:[]string{"192.168.174.134:2379"},
		DialTimeout:5*time.Second,
	}
	if client,err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	// 1.加锁(创建租约lease,自动续租KeepAlive确保租约不过期,
	// 拿着租约到etcd抢占1个key,抢到了key就是抢到了锁)
	// 抢key阶段租约是不能过期的
	lease = clientv3.NewLease(client)
	// 申请一个5秒租约
	if leaseGrantResp,err = lease.Grant(context.TODO(),5); err != nil {
		panic(err)
	}
	// 获取租约id
	leaseId = leaseGrantResp.ID

	// 自动续租
	// 用于取消自动续租的context 通过cancelFunc取消租约自动续租
	ctx,cancelFunc = context.WithCancel(context.TODO())
	// 确保main函数退出后自动续租会停止 租约过期后key会被etcd删除 终止自动续租的协程
	// 宕机 没有客户端给etcd服务端发送续租请求了 租约自动续租也会停止 key删除 锁释放
	defer cancelFunc()
	// NOTICE 希望释放锁的时候可以立即释放掉
	defer lease.Revoke(context.TODO(),leaseId)

	if leaseKeepAliveRespChan,err = lease.KeepAlive(ctx,leaseId); err != nil {
		panic(err)
	}

	// 开启一个协程 处理租约自动续租应答
	go func() {
		for {
			select {
			case leaseKeepAliveResp = <- leaseKeepAliveRespChan:
				if leaseKeepAliveResp == nil {
					// 通过context上下文取消续租 OR 网络异常
					// 则返回nil的RESP goto END 退出当前协程 自动续租结束
					fmt.Println("租约失效")
					goto END
				} else {
					fmt.Println("接收到租约自动续租应答,对应租约ID : ",leaseKeepAliveResp.ID)
				}
			}
		}
		END:
	}()

	// 拿租约到etcd抢占1个key
	// if 不存在key  then 设置该key  else 抢锁失败
	// 很多进程到etcd抢锁,谁第一个把key创建出,就是谁抢到锁了 其他人只能放弃
	// 当持有key的人把key释放,key删除,其他节点又可以抢这个key
	kv = clientv3.NewKV(client)
	// 创建事务txn context用来取消该事务txn
	txn = kv.Txn(context.TODO())
	// 判断key是否存在 CreateRevision是否0
	// "/cron/lock/job9" 的创建版本 是否等于 0 ; 是0则 该key不存在
	// clientv3.WithLease(leaseId) 防止节点宕机锁被永远占住
	// If 判断key是否被创建,如果key未被创建
	// Then 创建该key  满足If可以执行若干个Op
	// Else 如果key已被创建败则抢锁失败 不满足If可以执行另外若干个Op
	txn.If(clientv3.Compare(clientv3.CreateRevision("/cron/lock/job9"),"=",0)).
		Then(clientv3.OpPut("/cron/lock/job9","xxx",clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet("/cron/lock/job9"))
	// 提交事务txn
	if txnResp,err = txn.Commit(); err != nil {
		panic(err)
	}

	// 判断是否抢到了锁 进入THEN put key成功 进入ELSE失败
	// etcd 执行事务是原子性的 进入THEN的put一定是可以成功的
	if !txnResp.Succeeded{
		// 没提交成功就是 没抢到锁
		fmt.Println("锁被占用 : ",string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		return
		// 执行return语句 defer后的操作会被执行 租约会被释放
	}

	// 2.处理业务
	// 抢到锁,在分布式锁内,安全
	fmt.Println("抢到锁,在分布式锁内,安全 ... ")
	fmt.Println("处理任务...")
	// 模拟任务处理好久 占用锁
	time.Sleep(5 * time.Second)

	// 3.释放锁(取消租约自动续租,租约过期,释放key,
	// 等待租约过期释放锁 OR 直接释放租约,与租约关联的key马上被删除 )
	// 实现立即删除key的功能 (不是delete key 而是释放key关联的租约)
	// lease.KeepAlive(context,leaseId) 实现了自动续租
	// 需要中断 lease.KeepAlive 才能使租约过期 与租约关联的key删除 与key关联的锁才能释放
	//
	// 上面的
	// defer cancelFunc()
	// defer lease.Revoke(context.TO-DO(),leaseId)
	// 会把租约释放,租约关联的KV删除,相当于把锁释放掉
	//
	// 正常退出main也会执行defer后的操作 把锁释放掉
}

/*
[root@CentOS3 ~]# etcdctl watch "/cron/lock" --prefix
PUT
/cron/lock/job9
xxx
DELETE
/cron/lock/job9
*/

/*
终端1
$ go run etcd8-distribut-lock/main.go
接收到租约自动续租应答,对应租约ID :  7587841726481639909
抢到锁,在分布式锁内,安全 ...
处理任务...
接收到租约自动续租应答,对应租约ID :  7587841726481639909
接收到租约自动续租应答,对应租约ID :  7587841726481639909
 */

/*
终端2
$ go run etcd8-distribut-lock/main.go
接收到租约自动续租应答,对应租约ID :  7587841726481639912
锁被占用 :  xxx
*/
