package db

import (
	"fmt"

	"github.com/HarvestStars/gopool/protocol"
	"github.com/HarvestStars/gopool/server"
	"github.com/jinzhu/gorm"
)

// Setup 启动mysql配置
func Setup(user string, pwd string, host string, db string) {
	url := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, pwd, host, db)
	var err error
	server.DataBase, err = gorm.Open("mysql", url)
	if err != nil {
		panic("failed to connect database")
	}
	server.DataBase.AutoMigrate(&protocol.MinerInfo{})
}
