package specialdb

import (
	"errors"
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

//Transaction 统一事务
func Transaction(tx *gorm.DB,fn func()error)(err error){
	//开启事务
	//tx := db
	if tx.Error != nil {
		return errors.New("开启事务失败")
	}

	//todo 处理业务
	err = fn()
	if err != nil {
		errRb := tx.Rollback().Error
		if errRb != nil {
			return errors.New("事务回滚失败")
		}
		return err
	}

	//提交事务
	err = tx.Commit().Error
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}
