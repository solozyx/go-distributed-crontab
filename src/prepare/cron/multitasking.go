package main

import (
	"fmt"
	"time"

	"github.com/gorhill/cronexpr"
)

// 任务
type CronJob struct{
	expr *cronexpr.Expression
	// 任务是否过期
	nextTime time.Time
}

func main() {
	var (
		cronJob *CronJob
		expr *cronexpr.Expression
		now time.Time
		// 调度表 key 任务名称
		scheduleTable map[string]*CronJob
	)

	now = time.Now()
	// map 需要 make分配内存 无需设置容量cap 大小会自增长
	scheduleTable = make(map[string]*CronJob)

	// 定义2个CronJob
	// 每30秒执行1次
	expr = cronexpr.MustParse("*/30 * * * * * * ")
	cronJob = &CronJob{expr:expr,nextTime:expr.Next(now)}
	// 任务注册到调度表
	scheduleTable["job1"] = cronJob

	expr = cronexpr.MustParse("*/50 * * * * * * ")
	cronJob = &CronJob{expr:expr,nextTime:expr.Next(now)}
	scheduleTable["job2"] = cronJob

	// 开启1个调度协程,定时检查所有Cron任务,谁到期了就执行谁
	go func(){
		var (
			jobName string
			cronJob *CronJob
			now time.Time
		)
		// 定时检查任务调度表
		for {
			now = time.Now()
			for jobName,cronJob = range scheduleTable {
				// 判断cronJob是否到期 cronJob.nextTime <= now 说明到期 执行
				if cronJob.nextTime.Before(now) || cronJob.nextTime.Equal(now){
					// 开启1个协程 执行该 cronJob 外层调度协程 继续往下执行
					go func(jobName string){
						fmt.Println("执行: ",jobName)
					}(jobName)
				}
				// 计算下一次调度时间
				cronJob.nextTime = cronJob.expr.Next(now)
				fmt.Println(jobName," 下次执行时间: ",cronJob.nextTime)
			}
		}

		select {
		// 睡眠1000毫秒 进入下一轮外层调度 for轮询
		// .C 是一个channel 到了时间间隔向这个channel投递1个数据
		// select读这个.C队列channel 就会读到该数据
		// <-time.NewTimer(1000 * time.Millisecond).C 将在1000毫秒后可读,返回
		// 实现睡眠 1000 毫秒效果
		// timer的原理就是通过channel通知到期
		case <-time.NewTimer(1000 * time.Millisecond).C:

		}
	}()

	// 避免main协程退出
	time.Sleep(20*time.Second)
}