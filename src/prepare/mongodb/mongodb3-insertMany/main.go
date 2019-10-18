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
	// 任务执行开始时间 结束时间
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

		// interface 类似于 C的void*
		// 空接口可以装任何类型变量,空接口相当于通用的指针
		// 空指针 无类型 可以指向任意的内存地址
		// 指向的这块内存到底是什么类型 interface会记录它的type
		// interface 相当于记录了 (address,type)
		// address指向一块内存地址 type声明这块内存所存储数据类型 int double struct
		// JAVA的Object所有类的基类 可以把任何的对象转为Object
		// 内部也是一样的道理 指向一块内存address 记录内存中数据类型type
		// 通过记录的type可以转换为一个具体类型 反射上去
		logArr []interface{}

		insertManyResult *mongo.InsertManyResult
		// mongodb自增_id
		insertId interface{}
		//
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

	// mongodb存储的数据是bson
	record = &LogRecord{
		JobName:"job2",
		Command:"echo hello",
		Err:"",
		Content:"hello",
		TimePoint:TimePoint{
			StartTime:time.Now().Unix(),
			EndTime:time.Now().Unix()+10,
		},
	}

	// golang初始化容量
	// make
	// 或者在 {} 中直接定义有哪些元素
	logArr = []interface{}{record,record,record}

	// 4.insertMany
	// context
	// document []interface{}
	// opts 选项

	// 内部把record序列化为bson结构 结构体需要打 `bson` 标签
	if insertManyResult,err = collection.InsertMany(context.TODO(),logArr); err != nil {
		fmt.Println(err)
		return
	}

	// InsertedID 是空接口
	// mongo允许自定义主键_id
	// 如果不指定,_id就是mongo默认生成全局唯一 ObjectID:12字节二进制数组
	// interface{} -> ObjectID 反射出真实类型
	for _,insertId = range insertManyResult.InsertedIDs {
		// insertId interface{} -反射-> docId objectid.ObjectID
		docId = insertId.(objectid.ObjectID)
		fmt.Println("mongo 全局唯一自增ID : ",docId.Hex())
	}
}

/*
$ go run mongodb/mongodb3-insertMany/main.go
mongo 全局唯一自增ID :  5da95cd17fa5b2dda8879397
mongo 全局唯一自增ID :  5da95cd17fa5b2dda8879398
mongo 全局唯一自增ID :  5da95cd17fa5b2dda8879399
*/