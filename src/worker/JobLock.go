package worker

import (
	"github.com/coreos/etcd/clientv3"
	"context"
	"common"
	"fmt"
)

/*
分布式锁
在etcd中获取分布式锁,本质TXN事务抢etcd的key ,谁先抢到key谁就占到分布式锁
通过lease租约让key自动过期,避免worker宕机锁永远占用
worker宕机锁会自动过期,锁得到释放

不同的任务上不同的锁,不是全局锁，而是每个任务不能并发执行
不同任务之间可以并发,所以是锁到某个特定jobName任务
*/
type JobLock struct{
	kv clientv3.KV
	lease clientv3.Lease
	// 锁 jobName 任务
	jobName string
	// 用于终止自动续租 释放key 释放锁
	cancelFunc context.CancelFunc
	// 用于释放租约
	leaseId clientv3.LeaseID
	// 简化编程 增加 1个状态标识 是否上锁成功
	isLocked bool
}

/*
初始化锁
*/
func InitJobLock(jobName string,kv clientv3.KV,lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		jobName:jobName,
		kv:kv,
		lease:lease,
	}

	return
}

/*
尝试上锁:抢乐观锁 抢到就抢到 抢不到就算了
*/
func (jobLock *JobLock)TryLock()(err error){
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		// 用于取消lease的自动续租
		cancelCtx context.Context
		cancelFunc context.CancelFunc
		leaseKeepAliveResponseChan <- chan *clientv3.LeaseKeepAliveResponse
		txn clientv3.Txn
		// 锁路径 抢锁
		lockKey string
		txnResponse *clientv3.TxnResponse
	)
	// 1.创建lease租约 5秒 防止worker节点宕机后 让锁自动去释放掉 所以需要租约
	if leaseGrantResp,err = jobLock.lease.Grant(context.TODO(),5); err != nil {
		// 租约创建失败 终止
		return
	}
	// 2.自动续租 让lease租约不要过期 抢锁成功需要一直续租 不要让租约过期 租约过期锁会被释放
	// 想 unlock 的时候 任务处理完释放锁
	cancelCtx,cancelFunc = context.WithCancel(context.TODO())
	leaseId = leaseGrantResp.ID
	if leaseKeepAliveResponseChan,err = jobLock.lease.KeepAlive(cancelCtx,leaseId); err != nil {
		// 自动续租失败 直接return 租约会自动过期的
		// 续租失败 就是 抢分布式乐观锁失败
		goto FAIL
	}

	// 协程 处理续租应答
	go func(){
		var(
			leaseKeepAliveResponse *clientv3.LeaseKeepAliveResponse
		)
		// 轮询处理自动续租应答
		for {
			select {
			//自动续租应答 向etcd服务端报心跳 心跳成功 etcd服务端返回应答 在该协程消费心跳应答
			case leaseKeepAliveResponse = <- leaseKeepAliveResponseChan:
				// 返回空指针 说明自动续租被取消掉 释放锁cancel 异常cancel
				if leaseKeepAliveResponse == nil {
					// 自动续租失败 退出该协程
					goto END
				}
			}
		}
		END:
	}()

	// 3.创建TXN事务
	txn = jobLock.kv.Txn(context.TODO())
	// 抢到key就是抢到锁
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName

	// 4.TXN事务抢锁
	// lockKey == 0 说明 lockKey不存在
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey),"=",0)).
		// put 抢占 lockKey 携带lease即便worker节点宕机
		// 锁也会被释放掉 不持续占有lockKey锁 其他worker可以继续强该lockKey
		Then(clientv3.OpPut(lockKey,"",clientv3.WithLease(leaseId))).
		// lockKey != 0 说明 lockKey已经被其他worker抢到
		Else(clientv3.OpGet(lockKey))
	// 提交事务
	if txnResponse,err = txn.Commit(); err != nil {
		// 提交事务失败 lockKey 可能抢到 可能没抢到
		// 抢到了lockKey 被etcd写入了 但是etcd应答因为网络失败
		// 无法确认lockKey是否被抢到
		// 去 FAIL 释放租约 这样 lockKey 不论是否真实写入etcd 都会过期释放
		// 把租约释放掉 lease关联的lockKey马上被删掉
		// 确保事务写成功 写失败 都做失败回滚
		// 分布式场景最大问题: 应答失败,但是有没有在服务端生效不能确定,能做的是彻底回滚事务
		goto FAIL
	}

	// 5.抢锁成功返回,抢锁过程出现任何异常回滚释放租约,释放租约->租约过期->key过期->锁立刻释放
	// If..Then..Else 走了 Else分支 lockKey已经被占用
	if !txnResponse.Succeeded {
		err = common.ERR_LOCK_ALREADY_REQUIRET
		goto FAIL
	}
	// 抢锁成功
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc

	// 抢占lockKey成功 就不要走 FAIL 标签释放租约 提前返回
	jobLock.isLocked = true
	fmt.Println("worker JobLock 抢锁成功 : ",lockKey)
	return
FAIL:
	// 取消自动续租协程
	cancelFunc()
	// 释放租约
	jobLock.lease.Revoke(context.TODO(),leaseId)
	fmt.Println("worker JobLock 抢锁失败 : ",lockKey)
	return
}

/*
释放锁 把lease释放掉 把自动续租关掉
*/
func (jobLock *JobLock)UnLock(){
	// 只有锁成功才去释放锁
	if jobLock.isLocked {
		// 取消自动续租协程
		jobLock.cancelFunc()
		// 释放租约->租约关联lockKey删除->锁释放
		jobLock.lease.Revoke(context.TODO(),jobLock.leaseId)
		fmt.Println("worker JobLock 释放锁 : ", common.JOB_LOCK_DIR + jobLock.jobName)
	}
}