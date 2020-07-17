package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/HarvestStars/gopool/protocol"
)

var miner string
var nonce string
var dl float64
var height int

// simulate miner
func main() {
	nonce := "78968689"
	height = 2
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		miner = arg
		go loop(miner, nonce, height)
	}
	terminal := make(chan os.Signal)
	signal.Notify(terminal, os.Interrupt)
	for {
		select {
		case <-terminal:
			return
		}
	}
}

func loop(miner string, nonce string, height int) {
	for i := 100; i > 0; i-- {
		fmt.Printf("矿工 %s: 第 %d 轮 \n", miner, i)
		params := make([]interface{}, 0, 4)
		params = append(params, miner)
		params = append(params, nonce)
		params = append(params, float64(i))
		params = append(params, float64(height))

		Req := &protocol.Req{
			JSONRPC: "1.0",
			ID:      "curltest",
			Method:  "submitnonce",
			Params:  params,
		}
		byte, _ := json.Marshal(Req)
		postBytes := bytes.NewReader(byte)
		resp, err := http.Post("http://127.0.0.1:8080/", "application/json", postBytes)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
	}
}
