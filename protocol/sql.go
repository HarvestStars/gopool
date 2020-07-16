package protocol

import "github.com/jinzhu/gorm"

// MinerInfo 矿工提交的结果
type MinerInfo struct {
	gorm.Model
	Addr  string
	Nonce string
	// submitnonce rpc use float64 not uint64
	DL     float64
	Height float64
}

type BlockMined struct {
	gorm.Model
	Height   float64
	BlockID  string
	Miner    string
	CoinBase float64
}
