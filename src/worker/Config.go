package worker

import (
	"encoding/json"
	"io/ioutil"
)

var (
	G_config *Config
)

type Config struct {
	// etcd集群地址
	EtcdEndpoints []string `json:"etcdEndpoints"`
	// etcd连接超时
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	// mongodb地址
	MongodbUri string `json:"mongodbUri"`
	// mongo连接超时
	MongodbConnectTimeout int `json:"mongodbConnectTimeout"`
	// mongo log批次阈值
	MongodbLogBatchSize int `json:"mongodbLogBatchSize"`
	// mongo log批次自动提交时间间隔限定
	MongodbLogBatchCommitTimeout int `json:"mongodbLogBatchCommitTimeout"`
}

func InitConfig(masterConfFilePath string) (err error) {
	var (
		content []byte
		config  Config
	)
	if content, err = ioutil.ReadFile(masterConfFilePath); err != nil {
		return
	}
	if err = json.Unmarshal(content, &config); err != nil {
		return
	}
	G_config = &config
	return
}
