package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
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

// 查找条件
type FindByJobName struct {
	JobName string `bson:"jobName"`
}

func main() {
	var(
		client *mongo.Client
		err error
		database *mongo.Database
		collection *mongo.Collection
		findByJobName *FindByJobName
		// 游标
		cursor mongo.Cursor
		// mongodb读取回来的是bson,需要反序列化为LogRecord结构体
		// bson -> LogRecord
		record *LogRecord
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

	// context
	// filter 过滤条件 查找哪些记录，类型interface{} 意思是接收bson结构体
	// opts参数翻页 limit offset
	// collection.Find()

	// jobName==1
	findByJobName = &FindByJobName{
		// JobName:"job1",
		JobName:"job2",
	}
	if cursor,err = collection.Find(context.TODO(),findByJobName,findopt.Skip(0),findopt.Limit(10)); err != nil {
		fmt.Println(err)
		return
	}
	// 游标占了mongodb的连接资源要释放 to_do 不控制它超时 一直等待
	defer cursor.Close(context.TODO())

	// 从mongodb取出满足条件的下一条记录
	// context控制 Next方法调用 超时时间
	for cursor.Next(context.TODO()) {
		// 每次定义新的 LogRecord 指针,字段的默认值是类型的空值
		record = &LogRecord{}
		// 反序列化 bson -> LogRecord 根据bson标签做映射关系
		// 传入record指针,向record指针指向的对象做反序列化,把原有的空值覆盖
		if err = cursor.Decode(record);err != nil {
			fmt.Println(err)
			return
		}
		// record 是指针 打印 使用 取值符
		fmt.Println(*record)
	}
}

/*
$ go run mongodb/mongodb4-find/main.go
{job1 echo hello  hello {1571374778 1571374788}}

$ go run mongodb/mongodb4-find/main.go
{job2 echo hello  hello {1571380433 1571380443}}
{job2 echo hello  hello {1571380433 1571380443}}
{job2 echo hello  hello {1571380433 1571380443}}

*/