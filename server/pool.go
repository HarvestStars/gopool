package server

import (
	"log"
	"time"

	"github.com/HarvestStars/gopool/db"
	"github.com/HarvestStars/gopool/protocol"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var PoolAddrs []string

type BlockHeight int
type MinerName string

var BindMap map[BlockHeight]map[MinerName]string

func isBindingPool(poolAddrs []string, miner string) bool {
	var isIn bool = false
	for _, v := range poolAddrs {
		if v == miner {
			isIn = true
		}
	}
	return isIn
}

func isRegistered(cpy *gin.Context) (string, bool) {
	var miner string
	if v, ok := cpy.Request.Header["Account-Key"]; ok {
		miner := v[0]
		// redis 短链接
		RdsConn, err := db.RediShortConn(db.RdsHost, db.RdsPWD)
		if err != nil {
			log.Print("isRegistered: redis error", err.Error())
			return miner, false
		}

		poolAddress, err := redis.String(RdsConn.Do("get", "bind_"+miner))
		if err != nil {
			log.Print(err.Error())
			return miner, false
		}

		if isIn := isBindingPool(PoolAddrs, poolAddress); !isIn {
			return miner, false
		}
		return miner, true
	}
	return miner, false
}

func isBindingOnChain(miner string) (string, bool) {
	// lavad通信寻找被绑定地址
	to, err := getBindingInfo(miner)
	if err != nil {
		return "", false
	}

	// 检测被绑定地址是否在本地数据库中
	if isIn := isBindingPool(PoolAddrs, to); !isIn {
		return "", false
	}
	return to, true
}

func checkBindingMap(height int, address string) bool {
	// 绑定关系表查询
	if relation, ok := BindMap[BlockHeight(height)]; ok {
		if _, ok := relation[MinerName(address)]; !ok {
			to, isBinding := isBindingOnChain(address)
			if !isBinding {
				log.Printf("%s not binding to the pool", address)
				return false
			}
			relation[MinerName(address)] = to
		}
	} else {
		to, isBinding := isBindingOnChain(address)
		if !isBinding {
			log.Printf("%s not binding to the pool", address)
			return false
		}
		relationNew := make(map[MinerName]string)
		relationNew[MinerName(address)] = to
		BindMap[BlockHeight(height)] = relationNew
	}

	// 删除5个块以前的关系
	if height-5 > 0 {
		if _, ok := BindMap[BlockHeight(height-5)]; ok {
			delete(BindMap, BlockHeight(height-5))
		}
	}

	return true
}

// DBTerminal
var DBTerminal chan int = make(chan int)
var BestHeight int

// RecordCoinBase go协程启动
func RecordCoinBase(c chan int) {
	d := time.Duration(time.Second * 10)
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			lastBlock := &protocol.BlockMined{}
			db.DataBase.Model(&protocol.BlockMined{}).Last(lastBlock)
			BestHeight = int(lastBlock.Height)
			count, err := GetBlockCount()
			if err != nil {
				DBTerminal <- 1
			}
			if int(count) > BestHeight {
				for height := BestHeight + 1; height <= int(count); height++ {
					blockid, err := GetBlockHash(float64(height))
					if err != nil {
						DBTerminal <- 1
					}
					txid, err := GetBlockCoinBaseTXID(blockid)
					if err != nil {
						DBTerminal <- 1
					}
					address, coinbase, err := GetCoinBase(txid)
					if err != nil {
						DBTerminal <- 1
					}
					db.DataBase.Model(&protocol.BlockMined{}).Create(&protocol.BlockMined{Height: float64(height), BlockID: blockid, Miner: address, CoinBase: coinbase})
				}
			}

		case <-DBTerminal:
			db.DataBase.Close()
			log.Print("shut down BlockChain SQL DB")
			c <- 1
			return
		}
	}
}
