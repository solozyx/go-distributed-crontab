package master

import (
	"github.com/coreos/etcd/clientv3"
	"time"
)

var(
	// 单例
	G_jobMgr *JobMgr
)

/*
任务管理器
*/
type JobMgr struct{
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
	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
	)
	// 初始化配置
	config = clientv3.Config{
		// etcd 集群地址
		// Endpoints:[]string{""},
		Endpoints:G_config.EtcdEndpoints,
		// 超时时间
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	// 客户端连接到etcd服务器
	if client,err = clientv3.New(config); err != nil {
		return
	}
	// KV lease
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	// 赋值单例
	G_jobMgr = &JobMgr{
		client:client,
		kv:kv,
		lease:lease,
	}

	return
}