package common

import "encoding/json"

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