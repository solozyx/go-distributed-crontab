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
	// 当前时间
	now = time.Now()

	// 分 [1..59]
	// 时 [0..23]
	// 日 [1..31]
	// 月 [1..12]
	// 周几 [0..6]

	// 每分钟执行1次   "* * * * *"
	// 每5分钟执行1次  "*/5 * * * *"
	// 每6分钟执行1次  "*/6 * * * *"
	if expr,err = cronexpr.Parse("*/6 * * * *");err != nil{
		fmt.Println(err)
		return
	}
	// 下次调度时间
	nextTime = expr.Next(now)
	fmt.Println(now)
	fmt.Println(nextTime)

	expr = expr
}

/*
$ go run cronexpr1/main.go
2019-10-17 14:58:21.5007651 +0800 CST m=+0.004987601
2019-10-17 15:00:00 +0800 CST
*/