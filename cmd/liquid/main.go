package main

import (
	"log"

	"github.com/HarvestStars/gopool/db"
	"github.com/HarvestStars/gopool/protocol"
	"github.com/HarvestStars/gopool/server"
	"github.com/HarvestStars/gopool/setting"
	"github.com/jinzhu/gorm"
)

var LiquidDB *gorm.DB

func main() {
	// 解析配置文件
	setting.Setup()
	log.Print("Load setting done.")
	// 设置lavad服务ip
	server.LavadHost = setting.LavadSetting.Host
	// 读取清算数据库
	db.Setup(setting.MySQLSetting.User, setting.MySQLSetting.PassWord, setting.MySQLSetting.Host, setting.MySQLSetting.DataBase)
	log.Print("Liquid DB is open.")

	// 开始清算
	var currentHeightOnCache int = 0
	for {
		liquidHeight := &protocol.LiquidHeight{}
		db.DataBase.Model(&protocol.LiquidHeight{}).Last(liquidHeight)
		currentHeight, _ := server.GetBlockCount()
		if int(currentHeight) == currentHeightOnCache {
			continue
		}
		currentHeightOnCache = int(currentHeight)

		for h := liquidHeight.Height; h <= int32(currentHeight); h++ {
			log.Printf("Scan Height: %d.", h)
			h := int32(h)
			// confirmed block > 2
			if int32(currentHeight)-h > 2 {
				blkID, _ := server.GetBlockHash(float64(h))
				blockMined := &protocol.BlockMined{}
				db.DataBase.Model(&protocol.BlockMined{}).Where("height = ?", float64(h)).First(blockMined)
				if blockMined.BlockID != blkID {
					// it means the chain roll back
					db.DataBase.Model(&protocol.LiquidHeight{}).Create(&protocol.LiquidHeight{Height: h})
					continue
				}

				// lets liquid and record
				coinAllocated := blockMined.CoinBase
				var miners []protocol.MinerInfo
				var sum int
				db.DataBase.Model(&protocol.MinerInfo{}).Where("height = ?", float64(h)).Group("addr").Find(&miners)
				db.DataBase.Model(&protocol.MinerInfo{}).Where("height = ?", float64(h)).Count(&sum)
				if sum == 0 {
					db.DataBase.Model(&protocol.LiquidHeight{}).Create(&protocol.LiquidHeight{Height: h})
					continue
				}

				for i := 0; i < len(miners); i++ {
					var count int
					miner := miners[i]
					db.DataBase.Model(&protocol.MinerInfo{}).Where("height = ? AND addr = ?", float64(h), miner.Addr).Count(&count)
					benefit := coinAllocated * float64(count) / float64(sum)
					db.DataBase.Model(&protocol.LiquidInfo{}).Create(&protocol.LiquidInfo{Miner: miner.Addr, Height: h, Benefit: benefit})
				}
				db.DataBase.Model(&protocol.LiquidHeight{}).Create(&protocol.LiquidHeight{Height: h})
				log.Printf("Height: %d, liquid is ok.", h)
			}
		}
	}
}
