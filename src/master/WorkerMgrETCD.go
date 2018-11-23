package master

import (
	"github.com/coreos/etcd/clientv3"
	"time"
	"context"
	"common"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var(
	G_workerMgrETCD *WorkerMgrETCD
)

type WorkerMgrETCD struct{
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}

func InitWorkerMgrETCD()(err error){
	var(
		// etcd配置
		config clientv3.Config
		// etcd客户端
		client *clientv3.Client
		// kv
		kv clientv3.KV
		// lease租约
		lease clientv3.Lease
	)
	// 初始化etcd连接配置
	config = clientv3.Config{
		Endpoints:G_config.EtcdEndpoints,
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	// 建立etcd服务端连接
	if client,err = clientv3.New(config); err != nil {
		return
	}
	// 获取KV Lease子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	// 赋值单例
	G_workerMgrETCD = &WorkerMgrETCD{
		client:client,
		kv:kv,
		lease:lease,
	}
	return
}

/*

*/
func (workerMgrETCD *WorkerMgrETCD)ListWorkers()(workers []string,err error){
	var(
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		workerIP string
	)
	// 初始化worker节点切片 确保workers不是nil空指针
	workers = make([]string,0)
	if getResp,err = workerMgrETCD.kv.Get(
		context.TODO(),
		common.JOB_WORKER_DIR,
		clientv3.WithPrefix()); err != nil {
		return
	}
	for _,kvPair = range getResp.Kvs {
		// kvPair.Key = /cron/workers/192.168.10.10
		workerIP = common.ExtractWorkerIP(string(kvPair.Key))
		workers = append(workers,workerIP)
	}
	return
}