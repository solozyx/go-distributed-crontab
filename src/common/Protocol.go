package common

import (
	"encoding/json"
	"strings"
)

/*
定时任务
*/
type Job struct{
	// 任务名称
	Name string `json:"name"`
	// 执行shell命令
	Command string `json:"command"`
	// cron表达式
	CronExpr string `json:"cronExpr"`
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
	job *Job
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
		job:job,
	}
}