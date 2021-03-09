package tenantdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/keylock"
	"github.com/soedev/soelib/common/utils"
	"github.com/soedev/soelib/orm/driver/postgres"
	"github.com/soedev/soelib/orm/driver/sqlserver"
	"github.com/soedev/soelib/orm/grom"
	"github.com/soedev/soelib/orm/grom/logger"
	"log"
	"sync"
	"time"
)

//dbMap 数据源缓存列表
var dbMap sync.Map

func GetSQLDb(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	return getSQLDbWithOpt(tenantID, crmdb, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
}

func getSQLDbWithOpt(tenantID string, crmdb *gorm.DB, config *gorm.Config) (*gorm.DB, error) {
	teantDS := TenantDataSource{Db: crmdb}
	teantDataSource, err := teantDS.GetByTenantID(tenantID)
	if err != nil {
		return nil, err
	}
	dbName, server, port := utils.GetDBInfo(teantDataSource.URL, teantDataSource.DriverClassname)
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
	var dialect gorm.Dialector
	if teantDataSource.DriverClassname == "org.postgresql.ds.PGSimpleDataSource" {
		dialect = postgres.Open(fmt.Sprintf("host=%s user=%s port=%d dbname=%s sslmode=disable password=%s TimeZone=Asia/Shanghai",
			server, teantDataSource.UserName, port, dbName, password))
	} else {
		dialect = sqlserver.Open(fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d;encrypt=disable",
			server, teantDataSource.UserName, password, dbName, port))
	}

	db, err := gorm.Open(dialect, config)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if teantDataSource.MaxPoolSize == 0 {
		teantDataSource.MaxPoolSize = 10
	}
	if teantDataSource.PoolSize == 0 {
		teantDataSource.PoolSize = 2
	}
	if teantDataSource.ExpMinute == 0 {
		teantDataSource.ExpMinute = 5
	}
	sqlDB.SetConnMaxLifetime(teantDataSource.ExpMinute * time.Minute)
	sqlDB.SetMaxIdleConns(teantDataSource.PoolSize)
	sqlDB.SetMaxOpenConns(teantDataSource.MaxPoolSize)
	return db, nil
}

//GetDbFromMap 增加数据源到缓存
func GetDbFromMap(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		sqldb, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := sqldb.Ping(); err != nil {
			sqldb.Close()
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
		return db, nil
	}
	newDb, err := GetSQLDb(tenantID, crmdb)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func GetDbFromMapWithOpt(tenantID string, crmdb *gorm.DB, config *gorm.Config) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		//go senMsgToWx(tenantID, db.DB().Stats())
		sqldb, _ := db.DB()
		if err := sqldb.Ping(); err != nil {
			sqldb.Close()
			dbMap.Delete(tenantID)
			log.Println("移除数据源：", tenantID)
			newDb, err := getSQLDbWithOpt(tenantID, crmdb, config)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := getSQLDbWithOpt(tenantID, crmdb, config)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func senMsgToWx(teantId string, status sql.DBStats) {
	if status.Idle == 0 {
		errMsg := fmt.Sprintf("租户：%s 数据源出现异常   最大连接：%d,打开连接：%d，使用连接：%d，等待连接：%d", teantId, status.MaxOpenConnections, status.OpenConnections, status.InUse,
			status.WaitCount)
		log.Println(errMsg)
		utils.SendMsgToWorkWx(utils.DefaultRegChatID, errMsg, utils.WorkWxAPIPath, utils.WorkWxRestTokenStr)
	}
}

func Transaction(tx *gorm.DB, fn func() error) (err error) {
	//开启事务
	//tx := db
	if tx.Error != nil {
		return errors.New("开启事务失败")
	}
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
