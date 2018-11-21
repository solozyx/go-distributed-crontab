package worker

import (
	"common"
	"os/exec"
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
			// 分布式锁
			jobLock *JobLock
		)
		// shell执行结果
		result = &common.JobExecuteResult{
			ExecuteInfo:jobExecuteInfo,
			// 执行shell没有输出 则返回空切片
			Output:make([]byte,0),
		}
		// 初始化分布式锁
		jobLock = G_jobMgr.CreateJobLock(jobExecuteInfo.Job.Name)

		result.StartTime = time.Now()

		// 抢锁 抢到锁就执行下面的代码 没抢到锁跳过下面代码
		// 抢锁 TryLock 是网络操作
		err = jobLock.TryLock()
		// shell任务执行完成 释放锁 defer无论走哪个分支 最后都会释放锁
		// 只有锁成功 isLocked 才会真正释放锁 可以大胆defer
		defer jobLock.UnLock()

		if err != nil {
			result.Err = err
			// 刚启动抢锁失败就结束
			result.EndTime = time.Now()
		}else{
			// 抢锁成功后重置任务开始时间
			result.StartTime = time.Now()
			// 执行shell命令 /bin/bash -c "shell"
			// cmd = exec.CommandContext(context.TODO(),"/bin/bash","-c",jobExecuteInfo.Job.Command)
			cmd = exec.CommandContext(
				jobExecuteInfo.CancelCtx,
				"/bin/bash",
				"-c",
				jobExecuteInfo.Job.Command)
			// 执行cmd并捕获输出
			output,err = cmd.CombinedOutput()
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}
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