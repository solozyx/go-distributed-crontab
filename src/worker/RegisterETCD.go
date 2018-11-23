package worker

import (
	"github.com/coreos/etcd/clientv3"
	"time"
	"net"
	"common"
	"context"
)

var(
	G_registerETCD *RegisterETCD
)

/*
worker节点注册到etcd Registry
*/
type RegisterETCD struct {
	// worker节点本机IP
	localIPV4 string
	client *clientv3.Client
	kv clientv3.KV
	// 带lease租约让worker宕机后 /cron/workers/{IP} 自动过期从etcd中删除
	// 实时性比较好的服务注册与发现
	lease clientv3.Lease

}

func InitRegisterETCD()(err error){
	var(
		// 本机IPV4物理网卡
		localIpv4 string
		// etcd配置
		config clientv3.Config
		// etcd客户端
		client *clientv3.Client
		// kv
		kv clientv3.KV
		// lease租约
		lease clientv3.Lease
	)
	// 本机IPV4物理网卡
	if localIpv4,err = getLocalIpv4(); err != nil {
		return
	}

	// 初始化etcd连接配置
	config = clientv3.Config{
		Endpoints:G_config.EtcdEndpoints,
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	// 建立etcd服务端连接
	if client,err = clientv3.New(config); err != nil {
		return
	}
	// 获取KV Lease自己
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	// 赋值单例
	G_registerETCD = &RegisterETCD{
		localIPV4:localIpv4,
		client:client,
		kv:kv,
		lease:lease,
	}
	// worker注册到etcd
	go G_registerETCD.keepOnline()
	return
}

/*
获取本机IP
如果机器有多张网卡 遍历所有网卡 只获取第1个IP
*/
func getLocalIpv4()(ipv4 string,err error){
	var (
		// 本机所有网卡IP地址切片
		addrs []net.Addr
		addr net.Addr
		// ip地址
		ipNet *net.IPNet
		// 是否IP地址
		isIpNet bool
	)
	// 获取本机所有网卡
	if addrs,err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 遍历所有网卡 取第1个非localhost网卡
	// localhost是Linux系统的回环地址 虚拟网卡 需要获取物理网卡
	for _,addr = range addrs {
		// TODO NOTICE 类型断言 指针
		// addr IP地址 Unix socket地址
		// addr 需要反解断言 判断是否IP地址 ipv4 ipv6
		// addr 向 IP地址指针反解断言
		// 如果成功反解断言到IP地址 && 非localhost回环地址
		if ipNet,isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过ipv6
			// ipNet.IP.To4()转为ipv4
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				// 获取到第1个ipv4物理网卡 就返回这张网卡
				return
			}
		}
	}
	err = common.ERR_NO_LOCAL_IPV4_FOUND
	return
}

/*
worker注册到etcd
注册到 /cron/workers/{IP}
*/
func (registerETCD *RegisterETCD)keepOnline(){
	var(
		regKey string
		leaseGrantResp *clientv3.LeaseGrantResponse
		err error
		leaseKeepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeepAliveResp *clientv3.LeaseKeepAliveResponse
		cancelCtx context.Context
		cancelFuc context.CancelFunc
	)
	for{
		cancelFuc = nil

		// worker节点注册key /cron/workers/{IP}
		regKey = common.JOB_WORKER_DIR + registerETCD.localIPV4
		// 授权租约lease 在worker宕机10秒后该租约过期 regKey被删除
		if leaseGrantResp,err = registerETCD.lease.Grant(context.TODO(),10); err != nil {
			// worker节点创建lease失败 需要重试 要把worker节点的存活状态报上去
			// 中途出现任何错误 反复尝试 for
			// 跳到 RETRY 等待1秒后 重新进入for轮询走流程
			goto RETRY
		}
		// 在worker节点存活期间 得到lease自动续租 防止worker节点被etcd调度不到
		if leaseKeepAliveRespChan,err = registerETCD.lease.KeepAlive(context.TODO(),leaseGrantResp.ID); err != nil {
			goto RETRY
		}
		// 自动续租成功 注册到etcd
		cancelCtx,cancelFuc = context.WithCancel(context.TODO())
		// put regKey成功即可 不关心 putResp应答
		if _,err = registerETCD.kv.Put(cancelCtx,regKey,"",
			clientv3.WithLease(leaseGrantResp.ID)); err != nil {
			goto RETRY
		}
		// 处理 regKey 续租应答
		for {
			select{
			case leaseKeepAliveResp = <- leaseKeepAliveRespChan:
				if leaseKeepAliveResp == nil {
					// regKey 续租失败
					// worker节点和etcd之间网络不通 导致续租不上 等网络连通再去续租租约可能已经过期了
					// 走 RETRY 重走流程 申请新的租约
					goto RETRY
				}
			}
		}
	RETRY:
		// 睡眠1秒重新走for轮询整个注册 /cron/workers/{IP} 流程
		time.Sleep(1*time.Second)
		// 取消regKey自动续租
		if cancelFuc != nil {
			// registerETCD.kv.Put(cancelCtx,regKey,"", put失败取消自动续租
			cancelFuc()
		}
	}
}