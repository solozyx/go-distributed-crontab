package common

import "errors"

var(
	// 在var里定义常量
	ERR_LOCK_ALREADY_REQUIRET = errors.New("etcd lockKey 锁已被占用")
)