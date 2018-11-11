package common

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