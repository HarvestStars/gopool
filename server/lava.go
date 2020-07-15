package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"

	"github.com/HarvestStars/gopool/protocol"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
)

// lava接口数据库通信
var DataBase *gorm.DB
var RedisPWD string
var RdsConn redis.Conn
var RdsHost string
var RdsPWD string

// 负载均衡
var LavadHost []string
var MiningInfoIndex int = 0
var SubmitIndex int = 0

func getMiningInfo() (*protocol.Resp, error) {
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getmininginfo",
		Params:  []interface{}{},
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)

	// 负载均衡
	lavaServerCount := len(LavadHost)
	MiningInfoIndex++
	MiningInfoIndex = MiningInfoIndex % lavaServerCount

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

	// redis 短链接
	RdsConn, err := redis.Dial("tcp", RdsHost)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer RdsConn.Close()
	if _, err := RdsConn.Do("AUTH", RdsPWD); err != nil {
		fmt.Print("redis auth error \n", err.Error())
		panic("failed to connect redis")
	}

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

	if dl < bestDL || bestDL == 0 {
		// good dl, send to lavad
		reqByte, err := json.Marshal(&Req)
		if err != nil {
			fmt.Print(err.Error())
		}
		readByte := bytes.NewReader(reqByte)
		resp, err := http.Post(LavadHost[SubmitIndex], "application/json", readByte)
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

		// 负载均衡
		lavaServerCount := len(LavadHost)
		SubmitIndex++
		SubmitIndex = SubmitIndex % lavaServerCount

		// 解析Res，并设置最佳成绩
		// 被接受表示返回值数量 > 1
		if len(resMap) > 1 {
			// 写入redis
			dlInt := int64(dl)
			dlStr := strconv.FormatInt(dlInt, 10)
			RdsConn.Do("set", address, dlStr)

			// 写入mysql
			DataBase.Create(&protocol.MinerInfo{Addr: address, Nonce: nonce, DL: dl, Height: height})
		}
		return Res.Result, nil
	}
	return protocol.Accept{Accept: false}, nil
}
