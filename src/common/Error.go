package common

import "errors"

var(
	// 在var里定义常量
	ERR_LOCK_ALREADY_REQUIRET = errors.New("etcd lockKey 锁已被占用")

	ERR_NO_LOCAL_IPV4_FOUND = errors.New("未找到本地机器IPV4物理网卡")
)