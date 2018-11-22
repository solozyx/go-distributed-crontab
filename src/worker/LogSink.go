package worker

import (
	"github.com/mongodb/mongo-go-driver/mongo"
	"common"
	"context"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"time"
)

var(
	// Log存储全局单例
	G_logSink *LogSink
)

/*
写日志
*/
type LogSink struct{
	// mongo客户端
	client *mongo.Client
	// log表
	logCollection *mongo.Collection
	// Scheduler 和 LogSink 2个协程通信
	logChan chan *common.JobLog
	// time.AfterFunc(d,func) 另外协程执行的回调函数func给本协程发通知
	autoCommitChan chan *common.JobLogBatch
}

/*
初始化日志写入器
*/
func InitLogSink()(err error){
	var(
		client *mongo.Client
	)
	if client,err = mongo.Connect(
		context.TODO(),
		G_config.MongodbUri,
		clientopt.ConnectTimeout(time.Duration(G_config.MongodbConnectTimeout) * time.Millisecond));
		err != nil {
		return
	}
	// 选择db和collection
	G_logSink = &LogSink{
		client:client,
		// cron数据库的log表
		logCollection:client.Database("cron").Collection("log"),
		logChan:make(chan *common.JobLog,1000),
		autoCommitChan:make(chan *common.JobLogBatch,1),
	}
	// 启动mongodb处理协程
	go G_logSink.writeLoop()
	return
}

/*
消费日志队列协程
轮询logSink.logChan
*/
func (logSink *LogSink)writeLoop(){
	var(
		jobLog *common.JobLog
		// 当前批次
		jobLogBatch *common.JobLogBatch
		commitTimer *time.Timer
		// 超时批次
		timeoutBatch *common.JobLogBatch
	)
	for {
		select {
		case jobLog = <- logSink.logChan:
			// 1条 1条 日志插入mongodb 太慢 低吞吐
			// logSink.logCollection.InsertOne()
			// 每次插入需要完成mongodb请求的往返,网络环境差耗时可能花费很多时间
			if jobLogBatch == nil {
				jobLogBatch = &common.JobLogBatch{}
				// 1秒间隔自动提交本批次日志
				commitTimer = time.AfterFunc(
					time.Duration(G_config.MongodbLogBatchCommitTimeout) * time.Millisecond,

					// TODO NOTICE 2个协程之间发通知 chan投递事件
					// 到达该时间间隔执行该回调函数 该回调函数 在另外1个协程去执行 涉及到并发问题
					// 其他协程去提交batch
					// LogSink.writeLoop() 也在提交 batch 导致2个协程提交同一个batch 并发操作
					// 该回调函数 发一个通知给 LogSink.writeLoop() 把 LogSink.writeLoop() 做成串行处理
					//
					// func(){
					// 	// TODO NOTICE
					//	// jobLogBatch指针是 回调函数func(){}之外 LogSink.writeLoop()协程的指针
					//	// 会随时在 LogSink.writeLoop() 改掉
						// logSink.autoCommitChan <- jobLogBatch
					// },
					//

					// 回调函数的返回值又是一个回调函数
					// 传入 LogSink.writeLoop() 协程的 jobLogBatch
					// 定期器到期执行 func(){ logSink.autoCommitChan <- batch }
					// 把 LogSink.writeLoop() 协程的 jobLogBatch 投递到 logSink.autoCommitChan
					// 形参 batch 在 回调函数的 闭包上下文 与 LogSink.writeLoop() 协程的 jobLogBatch 指针 无关了
					func(batch *common.JobLogBatch) func() {
						return func(){
							logSink.autoCommitChan <- batch
						}
					}(jobLogBatch),
				)
			}
			jobLogBatch.JobLogs = append(jobLogBatch.JobLogs,jobLog)
			// job任务比较少 调度周期比较长 很难攒满 MongodbLogBatchSize 容量
			// 给一个时间约束作为不满该容量的优化 超过时间批次自动提交 不管多少条
			if len(jobLogBatch.JobLogs) >= G_config.MongodbLogBatchSize {
				// 写入本批次日志
				logSink.saveJobLogs(jobLogBatch)
				// 清空batch下次使用
				jobLogBatch = nil
				// 本批次的commitTimer定时器不一定能触发 取消定时器
				// 本批次接收到第1条JobLog创建本批次 jobLogBatch 同时启动 本批次的定时器commitTimer
				// 1秒后commitTimer执行回调函数
				// 但是在1秒内会有大量的JobLog 迅速到达阈值 马上写入mongo落盘 清空本批次 此时定时器就没用了
				// jobLogBatch.JobLogs已经落盘了
				commitTimer.Stop()
				// 定时器Stop时可能已经触发了 恰好在1秒的时间点 本批次满了 定时器也触发了 可能Stop不掉
				// 定时器的回调函数已经把本批次投递到 autoCommitChan
				// 此时 会 进入 case timeoutBatch = <- logSink.autoCommitChan:
				// logSink.saveJobLogs(jobLogBatch)
				// logSink.saveJobLogs(timeoutBatch)
				// 可能导致同一个批次 提交2次
				// TODO 判断 过期本次 是否依旧是 当前批次
			}
		case timeoutBatch = <- logSink.autoCommitChan:
			// 当前批次 jobLogBatch 产生时 产生 commitTimer 定时器
			// 1秒后 触发定时器的回调函数 会把 本批次 jobLogBatch 投递到 logSink.autoCommitChan
			// 该case就能监听到 	作为过期批次 写入mongo
			if timeoutBatch != jobLogBatch {
				// 当前JobLogBatch已经提交了 或者 进入了下一个批次
				// 跳过已经提交落盘的批次
				continue
			}
			logSink.saveJobLogs(timeoutBatch)
			jobLogBatch = nil
		}
	}
}

/*
私有方法
批量写入日志到mongodb
*/
func (logSink *LogSink)saveJobLogs(jobLogBatch *common.JobLogBatch){
	// 不判断是否写入成功 不关心
	logSink.logCollection.InsertMany(context.TODO(),jobLogBatch.JobLogs)
}

/*
Scheduler模块 和 LogSink模块 通信
*/
func (logSink *LogSink)Append(jobLog *common.JobLog){
	select {
	// logSink.logChan 没满就放进去了
	case logSink.logChan <- jobLog:
	// logSink.logChan 满了 上面的case后的channel操作会阻塞 就走default分支
	// Scheduler传过来的JobLog不要了
	default:
		// logSink.logChan 队列满了就丢弃
	}
}