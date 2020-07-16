package main

import (
	"github.com/HarvestStars/gopool/db"
	"github.com/HarvestStars/gopool/server"
	"github.com/HarvestStars/gopool/setting"
	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func main() {
	// 解析配置文件
	setting.Setup()
	// 设置清算数据库
	db.Setup(setting.MySQLSetting.User, setting.MySQLSetting.PassWord, setting.MySQLSetting.Host, setting.MySQLSetting.DataBase)
	// 设置redis
	db.RdsHost = setting.RdsSetting.Host
	db.RdsPWD = setting.RdsSetting.Password
	// 设置lavad服务ip
	server.LavadHost = setting.LavadSetting.Host
	// 设置pool服务信息
	server.PoolAddrs = setting.PoolSetting.Address
	poolHost := setting.PoolSetting.Host
	// 初始化绑定关系表
	server.BindMap = make(map[server.BlockHeight]map[server.MinerName]string)
	// 清理redis
	isClean := db.RediClearBest()
	if !isClean {
		panic("redis clean error, please check redis server is running.")
	}

	// 打开服务
	r := gin.Default()
	r.POST("/", server.MiningHandler)
	r.Run(poolHost)
}
