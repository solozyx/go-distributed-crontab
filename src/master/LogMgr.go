package master

import (
	"common"
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
)

var (
	G_logMgr *LogMgr
)

type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

func InitLogMgr() (err error) {
	var (
		client *mongo.Client
	)
	// 建立mongodb连接
	if client, err = mongo.Connect(
		context.TODO(),
		G_config.MongodbUri,
		clientopt.ConnectTimeout(time.Duration(G_config.MongodbConnectTimeout)*time.Millisecond));
		err != nil {
		return
	}
	G_logMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}
	return
}

/*
查看任务日志
*/
func (logMgr *LogMgr) ListLog(jobName string, skip int, limit int) (jobLogs []*common.JobLog, err error) {
	var (
		jobLogFilter *common.JobLogFilter
		jobLogSort   *common.SortJobLogByStartTime
		cursor       mongo.Cursor
		jobLog       *common.JobLog
	)
	// TODO NOTICE
	// 没有jobName对应的日志,Find()返回空的结果集 err是空的 jobLogs也是空的
	// 为了方便调用者 可以 len(jobLogs) 判断 不用判断jobLogs是否nil
	// 这里make出来一定不是nil 长度为 0
	jobLogs = make([]*common.JobLog, 0)

	// 查询mongodb过滤条件
	jobLogFilter = &common.JobLogFilter{
		JobName: jobName,
	}
	// 排序
	jobLogSort = &common.SortJobLogByStartTime{
		SortOrder: -1,
	}
	// 发起mongodb查询
	if cursor, err = logMgr.logCollection.Find(
		context.TODO(), jobLogFilter, findopt.Sort(jobLogSort),
		findopt.Skip(int64(skip)), findopt.Limit(int64(limit))); err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		// bson 反序列化 JobLog
		jobLog = &common.JobLog{}
		if err = cursor.Decode(jobLog); err != nil {
			// 写入mongodb的日志非法 忽略
			continue
		}
		jobLogs = append(jobLogs, jobLog)
	}
	return
}
