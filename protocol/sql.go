package protocol

import "github.com/jinzhu/gorm"

// MinerInfo 矿工提交的结果
type MinerInfo struct {
	gorm.Model
	Addr   string
	Nonce  string
	DL     float64
	Height float64
}
