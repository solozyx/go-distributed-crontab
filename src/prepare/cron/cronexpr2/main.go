package main

import (
	"fmt"
	"time"

	"github.com/gorhill/cronexpr"
)

func main() {
	var (
		expr *cronexpr.Expression
		err error
		now time.Time
		nextTime time.Time
	)

	// cronexpr 支持7个时间粒度
	// 秒 [0..59]
	// 分 [0..59]
	// 时 [0..23]
	// 日 [1..31]
	// 月 [1..12]
	// 周几 [0..6]
	// 年
	if expr,err = cronexpr.Parse("*/6 * * * * * *");err != nil{
		fmt.Println(err)
		return
	}

	// 当前时间
	now = time.Now()
	// 下次调度时间
	nextTime = expr.Next(now)
	// nextTime - now
	// time.NewTimer(nextTime.Sub(now))
	time.AfterFunc(nextTime.Sub(now), func() {
		fmt.Println("now = ",now)
		fmt.Printf("nextTime = %v 调度..\n",nextTime)
	})

	time.Sleep(10*time.Second)
}

/*
$ go run cronexpr2/main.go
now =  2019-10-17 15:18:31.3830036 +0800 CST m=+0.005982601
nextTime = 2019-10-17 15:18:36 +0800 CST 调度..
*/
