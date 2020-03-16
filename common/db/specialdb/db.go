package specialdb

import (
	"github.com/jinzhu/gorm"
	"time"
)

type DbConfig struct {
	DBInfo          string
	Dialect         string
	LogEnable       bool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

//Setup 初始化数据库
func ConnDB(config DbConfig) (*gorm.DB, error) {
	db, err := gorm.Open(config.Dialect, config.DBInfo)
	if err != nil {
		return nil, err
	}
	db.DB().SetMaxIdleConns(config.MaxIdleConns)       //最大空闲数
	db.DB().SetMaxOpenConns(config.MaxOpenConns)       //最大连接数
	db.DB().SetConnMaxLifetime(config.ConnMaxLifetime) //设置最大空闲时间，超过将关闭连接
	db.LogMode(config.LogEnable)
	return db, nil
}
