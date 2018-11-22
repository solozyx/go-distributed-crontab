package common

import (
	"encoding/json"
	"strings"
	"time"
	"github.com/gorhill/cronexpr"
	"context"
)

//-----------------------------
// common/Protocol
// 供 master worker 引用 跨包访问 结构体字段,首字母需要大写
//-----------------------------

/*
定时任务
*/
type Job struct{
	// 任务名称
	Name string `json:"name"`
	// 执行shell命令
	Command string `json:"command"`
	// cron表达式 计算任务到期执行时间
	CronExpr string `json:"cronExpr"`
}

/*
任务调度计划
记录每一个任务的下一次执行时间
*/
type JobSchedulePlan struct {
	// 要调度的任务
	Job *Job
	// 解析Job.CronExpr得到cronexpr表达式
	Expr *cronexpr.Expression
	// Job下次调度执行时间
	NextTime time.Time
}

/*
任务执行信息
*/
type JobExecuteInfo struct{
	Job *Job
	// 理论上job计划开始时间
	PlanTime time.Time
	// 真实job开始时间
	RealTime time.Time
	// 用于取消 /cron/jobs/jobName 对应 shell命令执行的上下文
	CancelCtx context.Context
	// 用于取消 /cron/jobs/jobName 对应 shell命令执行的方法
	CancelFunc context.CancelFunc
}

/*
job shell 执行结果
*/
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo
	// 执行shell命令的标准输出
	Output []byte
	// 执行shell命令的标准错误输出
	Err error
	// shell 脚本真实启动时间
	StartTime time.Time
	EndTime time.Time
}

/*
shell任务执行日志
*/
type JobLog struct{
	// 任务名
	JobName string `bson:"jobName"`
	// shell命令
	Command string `bson:"command"`
	// shell命令执行错误输出
	Err string `bson:"err"`
	// shell命令执行标准输出
	Output string `bson:"output"`
	// shell命令计划调度时间 Unix时间戳 精确到毫秒或微秒
	PlanTime int64 `bson:"planTime"`
	// shell命令真实调度时间 如果和 PlanTime相差大
	// 说明worker节点调度繁忙 导致来不及扫描所有任务
	// 正常是毫秒微秒时间差
	ScheduleTime int64 `bson:"scheduleTime"`
	// shell命令执行启动时间点
	StartTime int64 `bson:"startTime"`
	// shell命令执行结束时间点 和 StartTime 的差值表示执行任务时长
	EndTime int64 `bson:"endTime"`
}

/*
不要逐条写入JobLog到mongodb
提高吞吐量
*/
type JobLogBatch struct{
	JobLogs []interface{}
}

/*
master查询mongodb日志过滤条件
*/
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

/*
master查询mongodb日志排序条件
*/
type SortJobLogByStartTime struct {
	// 按JobLog开始时间倒序 {startTime:-1}
	SortOrder int `bson:"startTime"`
}

/*
http接口应答
*/
type Response struct {
	Errno int `json:"errno"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

/*
任务变化事件Event
1.更新Event
2.删除Event
*/
type JobEvent struct {
	// SAVE DELETE
	EventType int
	// SAVE的job DELETE的job
	Job *Job
}

/*
http接口应答API
*/
func BuildResponse(error int,msg string,data interface{})(resp []byte,err error){
	var(
		response Response
	)
	response.Errno = error
	response.Msg = msg
	response.Data = data
	resp,err = json.Marshal(&response)
	return
}

/*
json字符串 -> 反序列化为对象
*/
func UnpackJob(jsonStrByte []byte) (job *Job,err error){
	var(
		jobPtr *Job
	)
	// 创建1个Job对象
	jobPtr = &Job{}
	if err = json.Unmarshal(jsonStrByte,jobPtr); err != nil {
		return
	}
	job = jobPtr
	return
}

/*
从 /cron/jobs/job1 抹掉 /cron/jobs/ 提取 job1
*/
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey,JOB_SAVE_DIR)
}

/*
从 /cron/killer/job1 抹掉 /cron/killer/ 提取 job1
*/
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey,JOB_KILL_DIR)
}

func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType:eventType,
		Job:job,
	}
}

/*
构造任务执行计划
*/
func BuildJobSchedulePlan(job *Job)(jobSchedulePlan *JobSchedulePlan,err error){
	var(
		expr *cronexpr.Expression
	)
	// cron表达式解析
	if expr,err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	jobSchedulePlan = &JobSchedulePlan{
		Job:job,
		Expr:expr,
		NextTime:expr.Next(time.Now()),
	}
	return
}

/*
构造任务执行状态信息
*/
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan)(jobExecuteInfo *JobExecuteInfo){
	jobExecuteInfo = &JobExecuteInfo{
		Job:jobSchedulePlan.Job,
		// 计划调度时间
		PlanTime:jobSchedulePlan.NextTime,
		// 真实调度时间 计算繁忙导致延迟
		RealTime:time.Now(),
	}
	jobExecuteInfo.CancelCtx,jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}