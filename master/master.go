package main

import (
	"runtime"
	"master"
	"fmt"
	"flag"
	"time"
)

var (
	// 接收命令行参数
	masterConfFileParam string
)

func main() {
	var (
		err error
	)
	// 初始化命令行参数
	initArgs()
	// 初始化线程
	initEnv()
	// 加载master配置
	if err = master.InitConfig(masterConfFileParam); err != nil {
		goto ERR
	}
	// 初始化worker集群管理器
	if err = master.InitWorkerMgrETCD(); err != nil {
		goto ERR
	}
	// 读取mongo日志
	if err = master.InitLogMgr(); err != nil {
		goto ERR
	}
	// 启动etcd任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}
	// 启动API HTTP服务在协程里跑
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}
	// http服务在协程里跑 主函数马上退出 不让退出
	for{
		time.Sleep(1*time.Second)
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

/*
命令行参数
*/
func initArgs(){
	// flag库 解析命令行参数
	// 主节点启动 master -config ./master.json
	// *string 解析命令行传参
	// name 配置参数名 -config
	// value 默认值 命令行没有传入 -config 的 ./master.json 值的话使用默认值
	// usage 使用介绍 master -h 查看配置项的提示内容
	//
	flag.StringVar(&masterConfFileParam,"name","./master.json","传入master节点配置文件")
	// 解析命令行参数 赋值到 &masterConfFilePath
	//
	flag.Parse()
}