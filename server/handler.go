package server

import (
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/HarvestStars/gopool/protocol"
	"github.com/gin-gonic/gin"
)

// MinersChannelGroup 管理每个miner的redis最优解GET和SET
var MinersChannelGroup sync.Map

// MiningHandler is a midware for mining
func MiningHandler(c *gin.Context) {
	// 检验地址是否注册
	cpy := c.Copy()
	miner, isReg := isRegistered(cpy)
	if !isReg {
		log.Printf("%s does not regeistered", miner)
		return
	}

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
		v := reflect.ValueOf(miningReq.Params)
		arrayV := v.Interface().([]interface{})
		address := arrayV[0].(string)
		nonce := arrayV[1].(string)
		dl := arrayV[2].(float64)
		height := arrayV[3].(float64)

		if address != miner {
			// accountKey != miner in the miner.conf
			// miner wants to cheat me
			return
		}

		// 验证链上绑定关系
		checked := checkBindingMap(int(height), address)
		if !checked {
			return
		}

		// sync.Map 解决miner提交高并发的资源竞争问题
		newChan := make(chan int, 1)
		minerChan, ok := MinersChannelGroup.LoadOrStore(address, newChan)
		minerChanInt := minerChan.(chan int)
		if !ok {
			minerChanInt <- 1
		}
		res, err := submitNonce(miningReq.Params, address, nonce, dl, height, minerChanInt)
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
