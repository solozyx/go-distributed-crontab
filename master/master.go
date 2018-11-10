package main

import (
	"runtime"
	"master"
	"fmt"
)

func main() {
	var (
		err error
	)
	// 初始化线程
	initEnv()
	// 启动API HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}
	// 正常退出
	return
ERR:
	// 异常退出
	fmt.Println(err)
}

/*
设置golang运行线程 = 机器逻辑态CPU数量
*/
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}