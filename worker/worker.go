package main

// 所有的包都是相对于当前项目设置的GOPATH
import (
	"runtime"
	"fmt"
	"flag"
	"worker"
	"time"
)

var (
	workerConfFileParam string
)

func main() {
	var (
		err error
	)
	initArgs()
	initEnv()
	if err = worker.InitConfig(workerConfFileParam); err != nil {
		goto ERR
	}
	// 初始化worker cron任务执行器
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}
	// 初始化worker cron任务调度器
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}
	// 初始化worker JobMgr任务管理器
	if err = worker.InitJobMgr(); err != nil{
		goto ERR
	}
	for {
		time.Sleep(1 * time.Second)
	}
	// 正常退出
	return
ERR:
	// 异常退出
	fmt.Println(err)
}

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func initArgs(){
	// 从节点启动 worker -config ./worker.json
	// 解析命令行参数 赋值到 &masterConfFileParam
	flag.StringVar(&workerConfFileParam,"name","./worker.json","传入worker节点配置文件")
	flag.Parse()
}