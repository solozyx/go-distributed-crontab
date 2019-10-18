package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

type TimeBeforeCond struct {
	// 标签 bson 定义 key {$lt:timestamp}
	Before int64 `bson:"$lt"`
}

type DeleteCond struct {
	// {"timePoint.startTime":{$lt:timestamp}}
	BeforeCond TimeBeforeCond `bson:"timePoint.startTime"`
}

func main() {
	var(
		client *mongo.Client
		err error
		database *mongo.Database
		collection *mongo.Collection

		delCond *DeleteCond
		delResult *mongo.DeleteResult
	)
	// 1.建立连接
	// context可以用于取消连接,因为网络连接可能超时
	if client,err = mongo.Connect(context.TODO(),"mongodb://192.168.174.134:27017",clientopt.ConnectTimeout(5*time.Second)); err != nil {
		fmt.Println(err)
		return
	}

	// 2.选择数据库 cron
	database = client.Database("cron")
	// 3.选择数据表 log 表Collection会被自动创建
	collection = database.Collection("log")

	// 4.删除开始时间 <= 当前时间 的所有日志
	// $lt less than
	// delete({"timePoint.startTime":{"$lt":当前时间}})
	// 一般用地址做传递 避免值拷贝 大多数序列化库兼容指针和值 传2个类型进去都行
	delCond = &DeleteCond{BeforeCond:TimeBeforeCond{Before:time.Now().Unix()}}
	if delResult,err = collection.DeleteMany(context.TODO(),delCond); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("删除行数: ",delResult.DeletedCount)
}

/*
$ go run mongodb/mongodb5-delete/main.go
删除行数:  4
*/