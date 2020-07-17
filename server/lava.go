package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"github.com/HarvestStars/gopool/db"
	"github.com/HarvestStars/gopool/protocol"
	"github.com/gomodule/redigo/redis"
)

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
	reqByte, _ := json.Marshal(Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		log.Println(err.Error())
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

func submitNonce(Params []interface{}, address string, nonce string, dl float64, height float64) (interface{}, error) {
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "submitnonce",
		Params:  Params,
	}

	// redis 短链接
	RdsConn, err := db.RediShortConn(db.RdsHost, db.RdsPWD)
	if err != nil {
		log.Print("submitnonce: redis error", err.Error())
		return protocol.Accept{Accept: false}, nil
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
		del, err := redis.Bool(RdsConn.Do("DEL", "miner_best_"+address))
		if err != nil {
			log.Println("del action is failed", err)
		} else {
			log.Println("del action is:", del)
		}
	}
	bestDL, _ := redis.Float64(RdsConn.Do("get", "miner_best_"+address))

	if dl < bestDL || bestDL == 0 {
		// good dl, send to lavad
		reqByte, err := json.Marshal(Req)
		if err != nil {
			log.Print(err.Error())
		}
		readByte := bytes.NewReader(reqByte)
		resp, err := http.Post(LavadHost[SubmitIndex], "application/json", readByte)
		if err != nil {
			log.Print(err.Error())
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
			RdsConn.Do("set", "miner_best_"+address, dlStr)

			// 写入mysql
			db.DataBase.Model(&protocol.MinerInfo{}).Create(&protocol.MinerInfo{Addr: address, Nonce: nonce, DL: dl, Height: height})
		}
		return Res.Result, nil
	}
	return protocol.Accept{Accept: false}, nil
}

func getBindingInfo(miner string) (string, error) {
	// from miner get to
	params := make([]interface{}, 0)
	params = append(params, miner)
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getbindinginfo",
		Params:  params,
	}
	reqByte, err := json.Marshal(&Req)
	if err != nil {
		log.Print(err.Error())
	}
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		return "", errors.New("lavad server down")
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	respValue := reflect.ValueOf(Res.Result)
	resMap := respValue.Interface().(map[string]interface{})
	if to, ok := resMap["to"]; ok {
		toMap := to.(map[string]interface{})
		if toAddress, isIn := toMap["address"]; isIn {
			toAdrStr := toAddress.(string)
			return toAdrStr, nil
		}
	}
	return "", errors.New("miner not binding pool")
}

// GetBlockCount 获取最新块高
func GetBlockCount() (float64, error) {
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getblockcount",
		Params:  []interface{}{},
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		log.Println("GetBlockCount:", err.Error())
		return float64(0), err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	respValue := reflect.ValueOf(Res.Result)
	currentHeight := respValue.Interface().(float64)
	return currentHeight, nil
}

// GetBlockHash 找到对应高度的区块hash
func GetBlockHash(height float64) (string, error) {
	params := make([]interface{}, 0, 1)
	params = append(params, height)
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getblockhash",
		Params:  params,
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		log.Println("GetBlockHash:", err.Error())
		return "", err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	respValue := reflect.ValueOf(Res.Result)
	blockHash := respValue.Interface().(string)
	return blockHash, nil
}

// GetBlockCoinBaseTXID 获取某个区块中的coinbase txid
func GetBlockCoinBaseTXID(blockid string) (string, error) {
	params := make([]interface{}, 0, 1)
	params = append(params, blockid)
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getblock",
		Params:  params,
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		log.Println("GetBlockCoinBaseTXID:", err.Error())
		return "", err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	respValue := reflect.ValueOf(Res.Result)
	block := respValue.Interface().(map[string]interface{})
	txIDs, ok := block["tx"]
	if !ok {
		return "", errors.New("No txs in this block")
	}
	txValues := reflect.ValueOf(txIDs)
	coinBase := txValues.Index(0).Interface().(string)
	return coinBase, nil
}

// GetCoinBase 获取coinbase对应的rawtx
func GetCoinBase(txid string) (string, float64, error) {
	params := make([]interface{}, 0, 1)
	params = append(params, txid)
	params = append(params, true)
	Req := &protocol.Req{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getrawtransaction",
		Params:  params,
	}
	reqByte, _ := json.Marshal(&Req)
	readByte := bytes.NewReader(reqByte)
	resp, err := http.Post(LavadHost[MiningInfoIndex], "application/json", readByte)
	if err != nil {
		log.Println("GetCoinBase:", err.Error())
		return "", float64(0), err
	}
	defer resp.Body.Close()
	Res := &protocol.Resp{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, Res)
	respValue := reflect.ValueOf(Res.Result)
	rawTx := respValue.Interface().(map[string]interface{})
	vouts, ok := rawTx["vout"]
	if !ok {
		return "", float64(0), errors.New("No vout in this tx")
	}
	vout0 := reflect.ValueOf(vouts)
	vout0Map := vout0.Index(0).Interface().(map[string]interface{})
	coin := vout0Map["value"].(float64)
	scriptPubKey := vout0Map["scriptPubKey"].(map[string]interface{})
	addressSlice := scriptPubKey["addresses"]
	coinBaseMiner := reflect.ValueOf(addressSlice).Index(0).Interface().(string)
	return coinBaseMiner, coin, nil
}
