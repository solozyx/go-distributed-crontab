package master

import (
	"common"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"
)

var (
	// golang程序每个模块在全局都是一个单例
	// 单例用全局变量表示
	// 其他模块通过单例变量去访问
	// 指针变量 不赋值 默认nil 空指针
	// 本包的变量/方法供外包访问 需要 首字母大写
	// 单例对象
	G_apiServer *ApiServer
)

/*
http接口
*/
type ApiServer struct {
	httpServer *http.Server
}

/*
初始化HTTP服务
*/
func InitApiServer() (err error) {
	var (
		mux       *http.ServeMux
		listener  net.Listener
		httpSever *http.Server
		// web静态资源根目录
		staticDir http.Dir
		// 静态资源http回调
		staticHandler http.Handler
	)
	// 配置路由 有很多接口 每个接口功能不同 要路由
	// 创建路由对象
	mux = http.NewServeMux()
	// pattern string URL参数
	// handler func(ResponseWriter, *Request) 回调函数
	// 参数(应答,请求) 应答对象写入你需要返回的数据做应答
	// 浏览器请求该URL 该回调函数会被回调
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	// golang加载静态文件页面
	staticDir = http.Dir(G_config.WebRoot)
	// 路由handler是处理动态请求
	// 静态资源从磁盘读取 渲染到浏览器 net/http库提供内置handler
	// 无需像上面那样自己定义
	staticHandler = http.FileServer(staticDir)
	// mux接收到请求的url如 /index.html 会和路由表中所有的规则进行匹配
	// 这里 /index.html 和上面的路由规则都无法匹配 但是能匹配 / 最大匹配原则
	// 请求 /index.html 匹配到 /
	// StripPrefix 把 "/index.html" 的 "/" 抹掉 得到 "index.html" 交给 staticHandler
	// 相当于做一次转发
	mux.Handle("/", http.StripPrefix("/", staticHandler))

	// 启动监听端口
	// http是tcp服务,启动TCP监听
	// network 网络协议 tcp / udp 这里tcp协议绑定端口
	// address 监听端口地址任意网卡,本机所有网卡的某一个端口 ":"任意ip的8070端口
	// if listener,err = net.Listen("tcp",":8070"); err != nil {
	// 把服务端端口写死了，交给运维部署，想要更换端口，需要重新编译程序，不合理
	// 增加配置功能
	// if listener,err = net.Listen("tcp",":8070"); err != nil {
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}
	// 创建http服务
	httpSever = &http.Server{
		// 接口一般毫秒粒度超时控制,一个接口超过2000毫秒 2秒就认为服务端有异常
		// 正常接口都是 毫秒 微秒 级别返回
		// ReadTimeout:5 * time.Second,
		ReadTimeout: time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		// WriteTimeout:5 * time.Second,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		// 路由,转发,当http收到请求之后回调handler方法,根据请求的url遍历路由表找到匹配的回调函数
		// 把流量转发给匹配的路由函数
		// 代理模式
		Handler: mux,
	}
	// 赋值单例
	G_apiServer = &ApiServer{httpServer: httpSever}
	// 把 httPServer 跑到一个协程
	// 传入监听器 启动了服务端
	go httpSever.Serve(listener)
	return
}

/*
保存任务接口
内部实现 首字母小写 不暴露
job保存到etcd
job是浏览器ajax提交上来的
POST请求 :
	job = {"name":"job1","command":"echo hello","cronExpr":"* * * * *"}
golang http 服务端默认不会解析POST表单 解析耗费CPU 需要主动解析
应答: 0成功 非0失败
	正常应答 {"errno":0,"msg":"error info","data":{任意json...}}
	{任意json...} 用 interface{} 表示 空接口类型是万能容器 可以放任意对象
*/
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		// 序列化好的json串
		bytes []byte
	)
	// 1.主动解析浏览器POST表单
	if err = req.ParseForm(); err != nil {
		// 如果提交表单合法一般不会解析失败
		goto ERR
	}
	// 2.获取表单job字段
	postJob = req.PostForm.Get("job")
	// 3.反序列化postJob -> 结构体Job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	// 4. job -> JobMgr -> etcd
	// 传入Job结构体指针job
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	// 5. job保存到etcd成功返回 正常应答
	// NOTICE 这里 err == nil 是json序列化成功没有错误
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	// 6. job保存到etcd成功返回 异常应答
	// NOTICE 这里 err == nil 是json序列化成功没有错误
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

/*
删除任务接口
POST请求: 表单格式 a=1&b=2&c=3
	ip:8070/job/delete
	name=job1
*/
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)
	// 解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 要删除的任务名
	name = req.PostForm.Get("name")
	// etcd服务端删除key=name任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

/*
列举所有crontab任务 不翻页 一次性从etcd中取出
GET
*/
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		err     error
		bytes   []byte
	)
	if jobList, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", jobList); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

/*
控制所有worker节点 把正在执行的某个任务杀死
POST /job/kill name=job1
*/
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err   error
		name  string
		bytes []byte
	)
	// 解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	name = req.PostForm.Get("name")
	if err = G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err != nil {
		resp.Write(bytes)
	}
}

/*
从mongodb查询任务日志
GET ip:8070?name=job1&skip=0&limit=10
req.Form 	 get表单
req.PostForm post表单
*/
func handleJobLog(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		// 任务名称
		name string
		// 翻页参数 从第几条开始
		skipParam string
		skip      int
		// 翻页参数 限制返回多少条
		limitParam string
		limit      int
		// mongo返回日志切片
		jobLogs []*common.JobLog
		// http应答
		bytes []byte
	)
	// 解析GET参数
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 获取请求参数
	name = req.Form.Get("name")
	skipParam = req.Form.Get("skip")
	limitParam = req.Form.Get("limit")
	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitParam); err != nil {
		limit = 20
	}
	// mongo发起查询
	if jobLogs, err = G_logMgr.ListLog(name, skip, limit); err != nil {
		goto ERR
	}
	// 返回http正常应答
	if bytes, err = common.BuildResponse(0, "success", jobLogs); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	// 返回http异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

/*
master节点获取etcd集群健康worker节点列表
*/
func handleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		workers []string
		err     error
		bytes   []byte
	)
	if workers, err = G_workerMgrETCD.ListWorkers(); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", workers); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}
