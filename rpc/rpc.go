package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pool/protocol"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
)

var DataBase *gorm.DB
var RedisPWD string
var RdsConn redis.Conn
var mutex sync.Mutex

// MiningHandler is a midware for mining
func MiningHandler(c *gin.Context) {
	miningReq := &protocol.Req{}
	err := c.Bind(miningReq)
	if err != nil {
		return
	}
	switch miningReq.Method {
	case "getmininginfo":
		resp, err := getMiningInfo(miningReq.Params)
		c.JSON(200, gin.H{"result": resp.Result, "error": err, "id": resp.ID})
	case "submitnonce":
		res, err := submitNonce(miningReq.Params)
		c.JSON(200, gin.H{"result": res, "error": err, "id": "curltest"})
	case "async":
		fmt.Print("这是一次请求\n")
		time.Sleep(time.Duration(30) * time.Second)
		c.JSON(200, gin.H{"result": "time out", "error": nil, "id": "curltest"})
	default:
		return
	}
}

func getMiningInfo(Params []interface{}) (*protocol.Resp, error) {
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getmininginfo",
		Params:  Params,
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post("http://test:test@127.0.0.1:18332/", "application/json", readByte)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	return Res, nil
}

func submitNonce(Params []interface{}) (interface{}, error) {
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "submitnonce",
		Params:  Params,
	}
	v := reflect.ValueOf(Req.Params)
	arrayV := v.Interface().([]interface{})
	address := arrayV[0].(string)
	nonce := arrayV[1].(string)
	dl := arrayV[2].(float64)
	height := arrayV[3].(float64)

	//
	RdsConn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Print(err.Error())
	}
	defer RdsConn.Close()
	var pwd string = "123456"
	if _, err := RdsConn.Do("AUTH", pwd); err != nil {
		fmt.Print("redis auth error \n", err.Error())
		panic("failed to connect redis")
	}

	//mutex.Lock()
	bestHeight, _ := redis.Float64(RdsConn.Do("get", "best_height"))
	if int(bestHeight) == 0 {
		hInt := int64(height)
		hStr := strconv.FormatInt(hInt, 10)
		RdsConn.Do("set", "best_height", hStr)
	}
	if bestHeight != height {
		hInt := int64(height)
		hStr := strconv.FormatInt(hInt, 10)
		RdsConn.Do("set", "best_height", hStr)
		del, err := redis.Bool(RdsConn.Do("DEL", address))
		if err != nil {
			fmt.Println("del action is failed", err)
		} else {
			fmt.Println("del action is:", del)
		}
	}
	bestDL, _ := redis.Float64(RdsConn.Do("get", address))
	//mutex.Unlock()
	if dl < bestDL || bestDL == 0 {
		// good dl, send to lavad
		reqByte, err := json.Marshal(&Req)
		if err != nil {
			fmt.Print(err.Error())
		}
		readByte := bytes.NewReader(reqByte)
		resp, err := http.Post("http://test:test@127.0.0.1:18332/", "application/json", readByte)
		if err != nil {
			fmt.Print(err.Error())
			return protocol.Accept{Accept: false}, nil
		}
		defer resp.Body.Close()
		Res := &protocol.Resp{}
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, Res)
		respValue := reflect.ValueOf(Res.Result)
		resMap := respValue.Interface().(map[string]interface{})

		// 解析Res，并设置最佳成绩
		// 被接受表示返回值数量 > 1
		if len(resMap) > 1 {
			// 写入redis
			dlInt := int64(dl)
			dlStr := strconv.FormatInt(dlInt, 10)
			RdsConn.Do("set", address, dlStr)

			// 写入mysql 清算
			DataBase.Create(&protocol.MinerInfo{Addr: address, Nonce: nonce, DL: dl, Height: height})
		}
		return Res.Result, nil
	}
	return protocol.Accept{Accept: false}, nil
}
