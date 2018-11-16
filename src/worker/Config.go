package worker

import (
	"io/ioutil"
	"encoding/json"
)

var (
	G_config *Config
)

type Config struct {
	// etcd集群地址
	EtcdEndpoints []string `json:"etcdEndpoints"`
	// etcd连接超时
	EtcdDialTimeout int `json:"etcdDialTimeout"`

}

func InitConfig(masterConfFilePath string) (err error) {
	var (
		content []byte
		config Config
	)
	if content,err = ioutil.ReadFile(masterConfFilePath); err != nil {
		return
	}
	if err = json.Unmarshal(content,&config); err != nil {
		return
	}
	G_config = &config
	return
}