package common

const (
	// 任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	// 任务强杀目录
	JOB_KILL_DIR = "/cron/killer/"

	// etcd抢锁
	JOB_LOCK_DIR = "/cron/lock/"

	// worker注册到etcd
	JOB_WORKER_DIR = "/cron/workers/"

	// SAVE job 事件
	JOB_EVENT_SAVE = 1
	// DELETE job 事件
	JOB_EVENT_DELETE = 2
	// 强杀 job 事件
	JOB_EVENT_KILL = 3

)