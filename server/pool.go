package server

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var PoolAddrs []string

func isBindingPool(poolAddrs []string, miner string) bool {
	var isIn bool = false
	for _, v := range poolAddrs {
		if v == miner {
			isIn = true
		}
	}
	return isIn
}

func isRegistered(cpy *gin.Context) bool {
	if v, ok := cpy.Request.Header["Account-Key"]; ok {
		address := v[0]
		// redis 短链接
		RdsConn, err := redis.Dial("tcp", RdsHost)
		if err != nil {
			log.Print(err.Error())
			return false
		}
		defer RdsConn.Close()
		if _, err := RdsConn.Do("AUTH", RdsPWD); err != nil {
			log.Print(err.Error())
			return false
		}

		poolAddress, err := redis.String(RdsConn.Do("get", "bind__"+address))
		if err != nil {
			log.Print(err.Error())
			return false
		}

		if isIn := isBindingPool(PoolAddrs, poolAddress); !isIn {
			return false
		}
		return true
	}
	return false
}
