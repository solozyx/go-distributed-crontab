package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

type TimePoint struct {
	StartTime int64 `bson:"startTime"`
	EndTime int64 `bson:"endTime"`
}

type LogRecord struct {
	// 一般 `序列化类型json:"序列化后的字段名"`  JobName string `json:"job_name"`
	// mongo `序列化类型bson:"序列化后的字段名"`
	// 能序列化的字段名首字母必须大写 首字母小写是本包的私有字段无法导出

	// 任务名
	JobName string `bson:"jobName"`
	// shell命令
	Command string `bson:"command"`
	// 执行cron任务报错
	Err string `bson:"err"`
	// 执行cron任务输出
	Content string `bson:"content"`
	TimePoint TimePoint `bson:"timePoint"`
}

func main() {
	var(
		client *mongo.Client
		err error
		database *mongo.Database
		collection *mongo.Collection
		// LogRecord
		record *LogRecord
		insertOneResult *mongo.InsertOneResult
		docId objectid.ObjectID
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

	// 4.insertOne
	// context
	// document interface{} golang的空接口可以装任何类型变量,空接口相当于通用的指针
	// opts 选项
	// mongodb存储的数据是bson
	record = &LogRecord{
		JobName:"job1",
		Command:"echo hello",
		Err:"",
		Content:"hello",
		TimePoint:TimePoint{
			StartTime:time.Now().Unix(),
			EndTime:time.Now().Unix()+10,
		},
	}

	// 内部把record序列化为bson结构 结构体需要打 `bson` 标签
	if insertOneResult,err = collection.InsertOne(context.TODO(),record); err != nil {
		fmt.Println(err)
		return
	}

	// InsertedID 是空接口
	// mongo允许自定义主键_id
	// 如果不指定,_id就是mongo默认生成全局唯一 ObjectID:12字节二进制数组
	// interface{} -> ObjectID 反射出真实类型
	docId = insertOneResult.InsertedID.(objectid.ObjectID)
	fmt.Println("mongo 全局唯一自增ID : ",docId.Hex())
}

/*
$ go run mongodb/mongodb2-insertOne/main.go
mongo 全局唯一自增ID :  5da946ba9dee09406231b2d2

> show dbs
admin   0.000GB 系统库
config  0.000GB 系统库
cron    0.000GB
local   0.000GB 系统库
>
> use cron
switched to db cron
> show collections
log
> db.log.find()
{ "_id" : ObjectId("5da946ba9dee09406231b2d2"), "jobName" : "job1", "command" : "echo hello", "err" : "",
 "content" : "hello", "timePoint" : { "startTime" : NumberLong(1571374778), "endTime" : NumberLong(1571374788) } }
>
*/