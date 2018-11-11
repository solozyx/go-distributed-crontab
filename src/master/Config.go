package master

import (
	"io/ioutil"
	"encoding/json"
)

var (
	// 配置单例 其他模块直接访问单例配置项
	// 首字母大写可导出变量
	G_config *Config
)

/*
反序列化 master.json 获取配置参数
*/
type Config struct {
	ApiPort int `json:"apiPort"`
	ApiReadTimeout int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
}

/*
读取->反序列化->赋值单例
加载 master.json
json文件不合法则返回error
*/
func InitConfig(masterConfFilePath string) (err error) {
	var (
		content []byte
		config Config
	)
	// 1.读取配置文件 小文件 ioutil
	if content,err = ioutil.ReadFile(masterConfFilePath); err != nil {
		return
	}
	// 2.json反序列化 -> Config 结构体字段
	// json二进制数据
	// 反序列化目标结构体类型的指针
	if err = json.Unmarshal(content,&config); err != nil {
		return
	}
	// 3.赋值单例
	G_config = &config
	return
}