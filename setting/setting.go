package setting

import (
	"log"

	"github.com/go-ini/ini"
)

type RdsConf struct {
	Host        string `ini:"Host"`
	Password    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout int
}

var RdsSetting = &RdsConf{}

type MySQLConf struct {
	Host     string
	User     string
	PassWord string
	DataBase string
}

var MySQLSetting = &MySQLConf{}

type LavadConf struct {
	Host []string
}

var LavadSetting = &LavadConf{}

type PoolConf struct {
	Host    string
	Address []string
}

var PoolSetting = &PoolConf{}

// Setup 启动配置
func Setup() {
	cfg, err := ini.Load("../../conf/my.ini")
	if err != nil {
		log.Fatalf("Fail to parse 'conf/app.ini': %v", err)
	}
	mapTo(cfg, "redis", RdsSetting)
	mapTo(cfg, "mysql", MySQLSetting)
	mapTo(cfg, "lavad", LavadSetting)
	mapTo(cfg, "pool", PoolSetting)
}

func mapTo(cfg *ini.File, section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo RedisSetting err: %v", err)
	}
}
