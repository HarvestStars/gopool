package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-pool/protocol"
	"github.com/go-pool/rpc"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func main() {
	// 解析配置文件

	// 创建统计数据库
	var err error
	rpc.DataBase, err = gorm.Open("mysql", "root:123@tcp(localhost:3306)/lavapool?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic("failed to connect database")
	}
	defer rpc.DataBase.Close()
	rpc.DataBase.AutoMigrate(&protocol.MinerInfo{})

	// 打开服务
	r := gin.Default()
	r.POST("/", rpc.MiningHandler)
	r.Run(":8080")

}
