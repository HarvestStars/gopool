package server

import (
	"log"
	"time"

	"github.com/HarvestStars/gopool/protocol"
	"github.com/gin-gonic/gin"
)

// MiningHandler is a midware for mining
func MiningHandler(c *gin.Context) {
	// 检验地址是否注册
	cpy := c.Copy()
	if isReg := isRegistered(cpy); !isReg {
		return
	}

	// 检验地址绑定
	// TODO: 检验方式是与lavad通信，还是本地数据库通信

	// 获取body
	miningReq := &protocol.Req{}
	err := c.Bind(miningReq)
	if err != nil {
		log.Print(err.Error())
		return
	}

	// 服务
	switch miningReq.Method {
	case "getmininginfo":
		resp, err := getMiningInfo()
		c.JSON(200, gin.H{"result": resp.Result, "error": err, "id": resp.ID})
	case "submitnonce":
		res, err := submitNonce(miningReq.Params)
		c.JSON(200, gin.H{"result": res, "error": err, "id": "curltest"})
	case "async":
		log.Print("test async once.")
		time.Sleep(time.Duration(30) * time.Second)
		c.JSON(200, gin.H{"result": "time out", "error": nil, "id": "curltest"})
	default:
		log.Print("client invalid request.")
		return
	}
}
