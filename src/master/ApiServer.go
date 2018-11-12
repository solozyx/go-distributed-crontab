package master

import (
	"net/http"
	"net"
	"time"
	"strconv"
	"encoding/json"
	"common"
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
type ApiServer struct{
	httpServer *http.Server
}

/*
初始化HTTP服务
*/
func InitApiServer() (err error){
	var (
		mux *http.ServeMux
		listener net.Listener
		httpSever *http.Server
	)
	// 配置路由 有很多接口 每个接口功能不同 要路由
	// 创建路由对象
	mux = http.NewServeMux()
	// pattern string URL参数
	// handler func(ResponseWriter, *Request) 回调函数
	// 参数(应答,请求) 应答对象写入你需要返回的数据做应答
	// 浏览器请求该URL 该回调函数会被回调
	mux.HandleFunc("/job/save",handleJobSave)
	mux.HandleFunc("/job/delete",handleJobDelete)
	// 启动监听端口
	// http是tcp服务,启动TCP监听
	// network 网络协议 tcp / udp 这里tcp协议绑定端口
	// address 监听端口地址任意网卡,本机所有网卡的某一个端口 ":"任意ip的8070端口
	// if listener,err = net.Listen("tcp",":8070"); err != nil {
	// 把服务端端口写死了，交给运维部署，想要更换端口，需要重新编译程序，不合理
	// 增加配置功能
	// if listener,err = net.Listen("tcp",":8070"); err != nil {
	if listener,err = net.Listen("tcp",":" + strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}
	// 创建http服务
	httpSever = &http.Server{
		// 接口一般毫秒粒度超时控制,一个接口超过2000毫秒 2秒就认为服务端有异常
		// 正常接口都是 毫秒 微秒 级别返回
		// ReadTimeout:5 * time.Second,
		ReadTimeout:time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		// WriteTimeout:5 * time.Second,
		WriteTimeout:time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		// 路由,转发,当http收到请求之后回调handler方法,根据请求的url遍历路由表找到匹配的回调函数
		// 把流量转发给匹配的路由函数
		// 代理模式
		Handler:mux,
	}
	// 赋值单例
	G_apiServer = &ApiServer{httpServer:httpSever}
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
func handleJobSave(resp http.ResponseWriter, req *http.Request){
	var(
		err error
		postJob string
		job common.Job
		oldJob *common.Job
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
	if err = json.Unmarshal([]byte(postJob),&job); err != nil{
		goto ERR
	}
	// 4. job -> JobMgr -> etcd
	// 传入Job结构体指针job
	if oldJob,err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	// 5. job保存到etcd成功返回 正常应答
	// NOTICE 这里 err == nil 是json序列化成功没有错误
	if bytes,err = common.BuildResponse(0,"success",oldJob); err == nil{
		resp.Write(bytes)
	}
	return
ERR:
	// 6. job保存到etcd成功返回 异常应答
	// NOTICE 这里 err == nil 是json序列化成功没有错误
	if bytes,err = common.BuildResponse(-1,err.Error(),nil); err == nil{
		resp.Write(bytes)
	}
}

/*
删除任务接口
POST请求: 表单格式 a=1&b=2&c=3
	ip:8070/job/delete
	name=job1
*/
func handleJobDelete(resp http.ResponseWriter, req *http.Request)  {
	var(
		err error
		name string
		oldJob *common.Job
		bytes []byte
	)
	// 解析POST表单
	if err = req.ParseForm(); err != nil{
		goto ERR
	}
	// 要删除的任务名
	name = req.PostForm.Get("name")
	// etcd服务端删除key=name任务
	if oldJob,err = G_jobMgr.DeleteJob(name); err != nil{
		goto ERR
	}
	if bytes,err = common.BuildResponse(0,"success",oldJob); err == nil{
		resp.Write(bytes)
	}
	return
ERR:
	if bytes,err = common.BuildResponse(-1,err.Error(),nil); err == nil{
		resp.Write(bytes)
	}
}