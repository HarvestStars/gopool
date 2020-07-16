package db

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// redis接口
var RedisPWD string
var RdsConn redis.Conn
var RdsHost string
var RdsPWD string

// RediShortConn 创建redis短链接
func RediShortConn(host string, password string) (redis.Conn, error) {
	RdsConn, err := redis.Dial("tcp", host)
	if err != nil {
		log.Print(err.Error())
	}
	if _, err := RdsConn.Do("AUTH", password); err != nil {
		log.Print(err.Error())
	}
	return RdsConn, err
}

// RediClearBest 开启服务时首先清理redis
func RediClearBest() bool {
	conn, err := RediShortConn(RdsHost, RdsPWD)
	if err != nil {
		log.Print("redis connect error:", err.Error())
		return false
	}
	defer conn.Close()

	values, err := redis.Values(conn.Do("keys", "miner_best_*"))
	if err != nil {
		log.Print("redis clear error:", err.Error())
		return false
	}
	miners := make([]string, 0, 100)
	err = redis.ScanSlice(values, &miners)
	for _, v := range miners {
		conn.Do("del", v)
	}
	conn.Do("del", "best_height")
	log.Print("redis clean done")
	return true
}
