package common

import (
	"encoding/json"
	"strings"
	"time"
	"github.com/gorhill/cronexpr"
)

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