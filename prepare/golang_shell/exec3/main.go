package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type result struct {
	err error
	output []byte
}

func main() {
	// shell 命令执行很久可能发生异常 比如shell命令下载文件 把超时的任务杀死
	var(
		cmd *exec.Cmd
		// 命令与context有交互 ctx 里面有channel
		ctx context.Context
		// cancelFunc 做的事情就是把channel关掉了 close(channel)
		cancelFunc context.CancelFunc
		// 协程之间通信用 channel
		resultChan chan *result
		res *result
	)

	// 创建主线程执行结果队列
	resultChan = make(chan *result,1000)

	// 1. 执行1个cmd 让它在1个 goroutine 里去执行2秒 sleep 2;echo hello;
	// 2. 在shell命令执行1秒的时候杀死 cmd 让任务无法输出 hello

	// 开启1个协程 调用1个匿名函数
	// 这个函数会被扔到1个新协程去运行 main继续往下执行 并发
	// context.WithCancel()创建1个上下文 + 1个取消函数
	ctx,cancelFunc = context.WithCancel(context.TODO())

	go func(){
		var (
			output []byte
			err error
		)
		// 上下文 golang标准库
		// cmd对象内部维护了ctx对象
		// select {case ctx.Done():} 监听ctx内部的channel是否被关闭 会实时感知到
		// 调用了 cancelFunc 就把ctx内部channel关闭了
		// ctx的channel关闭了
		// cmd内部的select语法匹配到这个情况 就调用系统的kill pid进程ID.
		// 杀死子进程
		cmd = exec.CommandContext(ctx,"/bin/bash","-c","sleep 2;echo hello;")
		// 子进程执行shell命令 捕获任务输出
		// 子进程执行1秒 在main协程杀死该子协程 就返回到这里 拿到 输出+错误
		output,err = cmd.CombinedOutput()
		// 把执行任务结果传给main协程
		resultChan <- &result{
			output:output,
			err:err,
		}
	}()

	// main 函数在原来的线程/协程执行
	// main 协程 和 上面的go协程并发执行
	time.Sleep(1*time.Second)

	// 1秒后取消上下文 即 杀掉go子进程执行 子进程无法执行 echo hello;
	cancelFunc()

	// 在main协程 等待子协程退出 打印主线程任务执行结果
	// 阻塞在 <- resultChan 等待子协程执行完成 把执行结果投递进resultChan
	// 投递进去就取出来
	res = <- resultChan
	fmt.Println(res.err,string(res.output))
}