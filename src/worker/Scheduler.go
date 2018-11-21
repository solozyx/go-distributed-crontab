package worker

import (
	"common"
	"time"
	"fmt"
)

var(
	G_scheduler *Scheduler
)

/*
任务调度
不停轮询,检查哪些cron定时任务到期
实现稍微复杂
*/
type Scheduler struct{
	// channel etcd任务事件队列 任务变化的时候通知过来
	jobEventChan chan *common.JobEvent
	// job调度计划表 任务名全局唯一
	jobPlanTable map[string]*common.JobSchedulePlan
	// job执行表存放当前正在执行的任务
	jobExecutingTable map[string]*common.JobExecuteInfo
	// Executor执行shell结果回传Scheduler 任务执行结果队列
	jobExecuteResultChan chan *common.JobExecuteResult
}

/*
初始化调度器
*/
func InitScheduler()(err error){
	G_scheduler = &Scheduler{
		// make 1000 的容量
		jobEventChan:make(chan *common.JobEvent,1000),
		jobPlanTable:make(map[string]*common.JobSchedulePlan),
		jobExecutingTable:make(map[string]*common.JobExecuteInfo),
		jobExecuteResultChan:make(chan *common.JobExecuteResult,1000),
	}
	// 启动调度协程
	go G_scheduler.scheduleLoop()
	return
}

/*
启动调度协程
轮询cron定时任务是否到期
*/
func (scheduler *Scheduler)scheduleLoop(){
	var(
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobExecuteResult *common.JobExecuteResult
	)
	// 初始化(1秒 刚启动没有任务)
	scheduleAfter = scheduler.TrySchedule()
	// 调度延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	// for轮询监听任务事件
	for{
		select {
		// worker启动获取 全量任务 同步给 Scheduler
		// 在 scheduleLoop 死循环轮询 监听 scheduler.jobEventChan
		// 监听任务变化事件 当有任务来的时候 就匹配case
		case jobEvent = <-scheduler.jobEventChan:
			// 对内存中维护的job进行CRUD
			scheduler.handleJobEvent(jobEvent)
		case <- scheduleTimer.C :
			// 最近的任务到期
		// TODO Executor执行完shell把结果回传Scheduler,调度协程需要监听Executor的回传数据
		case jobExecuteResult = <- scheduler.jobExecuteResultChan:
			scheduler.handleJobResult(jobExecuteResult)
		}
		scheduleAfter = scheduler.TrySchedule()
		// 重置定时器
		scheduleTimer.Reset(scheduleAfter)
	}
}

/*
推送任务变化事件
*/
func (scheduler *Scheduler)PushJobEvent(jobEvent *common.JobEvent){
	scheduler.jobEventChan <- jobEvent
}

/*
处理任务事件
重点:维护scheduler任务列表,实时同步job任务,保持内存中任务和etcd保持一致
*/
func (scheduler *Scheduler)handleJobEvent(jobEvent *common.JobEvent){
	var(
		jobSchedulePlan *common.JobSchedulePlan
		err error
		jobExisted bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan,err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			// 解析cron表达式失败 静默处理
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		// etcd是有序的事件推送 一般job是存在的
		// etcd删除不存在的key还是会有delete事件过来 但是内存中已经没有该job了
		if jobSchedulePlan,jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted{
			delete(scheduler.jobPlanTable,jobEvent.Job.Name)
		}
	}
}

/*
重新计算任务调度状态
*/
func (scheduler *Scheduler)TrySchedule()(scheduleAfter time.Duration){
	var(
		jobPlan *common.JobSchedulePlan
		now time.Time
		// 初始化是nil空指针
		nearTime *time.Time
	)
	// 如果 scheduler.jobPlanTable 为空 随便睡眠多久
	if len(scheduler.jobPlanTable) == 0 {
		return 1 * time.Second
	}

	now = time.Now()
	// 1.遍历所有任务
	for _,jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			// TODO 任务到期 尝试(任务到期但是前一次执行还没有结束 不一定能启动它,要等前一次执行结束)执行任务
			// fmt.Println("worker scheduler 执行任务 : ",jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			// 基于当前时间 更新job下一次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		// 统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime){
			nearTime = &jobPlan.NextTime
		}
	}
	// 2.统计最近到期任务 scheduleAfter秒后到期 就 sleep scheduleAfter
	// 下次调度间隔 最近要执行的任务调度时间 - 当前时间
	scheduleAfter = (*nearTime).Sub(now)
	return
}

/*
调度:定期检查哪些任务到期
执行:发现到期任务尝试执行job任务
	 执行的任务可能运行很久,比如运行1分钟,1分钟调度60次,但是1分钟只能执行1次,
	 程序的定时器 time.timer 是不可能完全精准的
	 TODO 通过 jobExecutingTable 去重防止并发
	 所以是尝试启动任务
*/
func (scheduler *Scheduler)TryStartJob(jobPlan *common.JobSchedulePlan){
	var(
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
	)
	// 如果job正在执行跳过本次调度
	if jobExecuteInfo,jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		// job正在运行 静默处理
		fmt.Println("worker Scheduler 任务正在执行跳过本次执行 : ",jobPlan.Job.Name)
		// delete(scheduler.jobExecutingTable,jobExecuteResult.ExecuteInfo.Job.Name) 保证不走到此分支
		return
	}
	// job没有执行 则执行job任务
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo
	// TODO 执行job 启动shell命令 并发调度shell
	fmt.Println("worker Scheduler 开始执行job任务 : ",jobPlan.Job.Name,jobExecuteInfo.PlanTime,jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}

/*
job shell执行结果回调方法
*/
func (scheduler *Scheduler)PushJobResult(jobExecuteResult *common.JobExecuteResult){
	scheduler.jobExecuteResultChan <- jobExecuteResult
}

/*
Executor执行完shell把结果回传Scheduler,调度协程处理Executor的回传数据
*/
func (scheduler *Scheduler)handleJobResult(jobExecuteResult *common.JobExecuteResult){
	fmt.Println("worker Executor 回传 Scheduler 任务执行结果 : ",
		jobExecuteResult.ExecuteInfo.Job.Name,
		string(jobExecuteResult.Output),
		jobExecuteResult.Err)
	// 从正在执行的任务列表中删除该任务 保证后续该任务在 TrySchedule 能再次被调度执行
	delete(scheduler.jobExecutingTable,jobExecuteResult.ExecuteInfo.Job.Name)
}