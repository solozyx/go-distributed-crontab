package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

func main() {
	var(
		client *mongo.Client
		err error
		database *mongo.Database
		collection *mongo.Collection
	)
	// 1.建立连接
	// context可以用于取消连接,因为网络连接可能超时
	if client,err = mongo.Connect(context.TODO(),
		"mongodb://192.168.174.134:27017",clientopt.ConnectTimeout(5*time.Second)); err != nil {
		fmt.Println(err)
		return
	}
	// 2.选择数据库
	database = client.Database("my_db")
	// 3.选择数据表
	collection = database.Collection("my_collection")

	collection = collection
}