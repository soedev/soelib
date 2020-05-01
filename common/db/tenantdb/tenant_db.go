package tenantdb

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/keylock"
	"github.com/soedev/soelib/common/utils"
	"log"
	"sync"
	"time"
)

//dbMap 数据源缓存列表
var dbMap sync.Map

func GetSQLDb(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	teantDS := TenantDataSource{Db: crmdb}
	teantDataSource, err := teantDS.GetByTenantID(tenantID)
	if err != nil {
		return nil, err
	}
	dbName, server, port := utils.GetDBInfo(teantDataSource.URL)
	if dbName == "" {
		return nil, errors.New("数据源设置错误，数据库名为空！")
	}
	if server == "" {
		return nil, errors.New("数据源设置错误，数据库服务器！")
	}
	// DES 解密
	data := []byte(teantDataSource.Password)
	password := des.DecryptDESECB(data, des.DesKey)
	if password == "" {
		return nil, errors.New("数据源设置错误，密码为空！")
	}
	dbInfo := fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d;encrypt=disable", server, teantDataSource.UserName, password, dbName, port)
	sqlDb, err := gorm.Open("mssql", dbInfo)
	if err != nil {
		return nil, err
	}
	sqlDb.DB().SetMaxIdleConns(2)                  //最大空闲数
	sqlDb.DB().SetMaxOpenConns(10)                  //最大连接数
	sqlDb.DB().SetConnMaxLifetime(time.Minute * 5) //设置最大空闲时间，超过将关闭连接
	sqlDb.LogMode(false)
	return sqlDb, nil
}

//GetDbFromMap 增加数据源到缓存
func GetDbFromMap(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		if err := sqldb.(*gorm.DB).DB().Ping(); err != nil {
			dbMap.Delete(tenantID)
			log.Println("移除数据源：", tenantID)
			newDb, err := GetSQLDb(tenantID, crmdb)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return sqldb.(*gorm.DB), nil
	}
	newDb, err := GetSQLDb(tenantID, crmdb)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	log.Println("增加数据源：", tenantID)
	return newDb, nil
}
