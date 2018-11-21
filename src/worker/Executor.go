package worker

import (
	"common"
	"os/exec"
	"context"
	"time"
)

var(
	// 全局shell执行器单例
	G_executor *Executor
)

/*
shell命令执行器
*/
type Executor struct {

}

/*
执行shell命令
启动1个协程 很多job的shell任务要执行,并发去调度
*/
func (executor *Executor)ExecuteJob(jobExecuteInfo *common.JobExecuteInfo){
	go func() {
		var(
			cmd *exec.Cmd
			output []byte
			err error
			result *common.JobExecuteResult
		)
		// shell执行结果
		result = &common.JobExecuteResult{
			ExecuteInfo:jobExecuteInfo,
			// 执行shell没有输出 则返回空切片
			Output:make([]byte,0),
		}
		result.StartTime = time.Now()

		// 执行shell命令 /bin/bash -c "shell"
		cmd = exec.CommandContext(context.TODO(),"/bin/bash","-c",jobExecuteInfo.Job.Command)
		// 执行cmd并捕获输出
		output,err = cmd.CombinedOutput()
		result.EndTime = time.Now()
		result.Output = output
		result.Err = err

		// shell执行完成 把执行结果 返回给 Scheduler模块
		// Scheduler 从 jobExecutingTable 删除该任务
		// 下一次该任务到期后 又会再次执行
		// 如果不删除 jobExecutingTable 那么里面的任务只能执行1次
		// TODO 不同的协程通信用 channel Executor执行shell结果回传Scheduler 2个协程间通信
		G_scheduler.PushJobResult(result)
		// TODO Scheduler调度协程需要监听 Executor的回传
	}()
}

/*
初始化执行器
*/
func InitExecutor()(err error){
	// 赋值全局单例
	G_executor = &Executor{

	}
	return
}