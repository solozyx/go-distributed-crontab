package master

import (
	"net/http"
	"net"
	"time"
	"strconv"
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
*/
func handleJobSave(w http.ResponseWriter, r *http.Request){

}