package main

import (
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

func main() {
	var (
		// 客户端配置
		config clientv3.Config
		// 客户端指针
		client *clientv3.Client
		err error
	)
	config = clientv3.Config{
		// 连接etcd集群 有多个节点就配置多个ip 保证高可用
		Endpoints:[]string{"192.168.174.134:2379"},
		// 连接超时时间
		DialTimeout:5*time.Second,
	}
	// 建立连接
	if client,err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("etcd server connection success")
	client = client
}
