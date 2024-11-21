package tenantdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/keylock"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/common/utils"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"sync"
	"time"
)

// dbMap 数据源缓存列表
var dbMap sync.Map

type OptSQL struct {
	DBConfig        gorm.Config
	ApplicationName string
}

func GetSQLDb(tenantID string, crmDB *gorm.DB, enable bool) (*gorm.DB, error) {
	opt := &OptSQL{
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logger.Error)},
		ApplicationName: "go-saas-service",
	}
	return getSQLDbWithOpt(tenantID, crmDB, opt, enable)
}

func getSQLDbWithOpt(tenantID string, crmDB *gorm.DB, opt *OptSQL, enable bool) (*gorm.DB, error) {
	var tenantDataSource TenantDataSource
	var err error
	isNew := enable
	if enable {
		repository := NewTDSBRepository(crmDB)
		tenantDataSourceBack, err := repository.GetByTenantID(tenantID)
		if err != nil {
			return nil, err
		}
		if tenantDataSourceBack != nil && tenantDataSourceBack.Enable == 1 {
			utils.CopyStruct(tenantDataSourceBack, &tenantDataSource)
			if tenantDataSource.URL == "" {
				soelog.Logger.Error("备用数据源赋值失败")
			}
		} else {
			isNew = false
		}
	}
	if !isNew || tenantDataSource.URL == "" {
		repository := NewTDSRepository(crmDB)
		tenantDataSource, err = repository.GetByTenantID(tenantID)
		if err != nil {
			return nil, err
		}
	}
	dbName, server, port := utils.GetDBInfo(tenantDataSource.URL, tenantDataSource.DriverClassname)
	if dbName == "" {
		return nil, errors.New("数据源设置错误，数据库名为空！")
	}
	if server == "" {
		return nil, errors.New("数据源设置错误，数据库服务器！")
	}
	// DES 解密
	data := []byte(tenantDataSource.Password)
	password := des.DecryptDESECB(data, des.DesKey)
	if password == "" {
		return nil, errors.New("数据源设置错误，密码为空！")
	}
	dbInfo := fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d;encrypt=disable", server, tenantDataSource.UserName, password, dbName, port)
	sqlDb, err := gorm.Open(sqlserver.Open(dbInfo), &opt.DBConfig)
	if tenantDataSource.DriverClassname == "org.postgresql.ds.PGSimpleDataSource" {
		dbInfo = fmt.Sprintf("host=%s user=%s port=%d dbname=%s sslmode=disable password=%s application_name=%s",
			server, tenantDataSource.UserName, port, dbName, password, opt.ApplicationName)
		sqlDb, err = gorm.Open(postgres.Open(dbInfo), &opt.DBConfig)
	}
	if err != nil {
		return nil, err
	}
	if tenantDataSource.MaxPoolSize == 0 {
		tenantDataSource.MaxPoolSize = 10
	}
	if tenantDataSource.PoolSize == 0 {
		tenantDataSource.PoolSize = 2
	}
	if tenantDataSource.ExpMinute == 0 {
		tenantDataSource.ExpMinute = 5
	}
	db, err := sqlDb.DB()
	if err != nil {
		return nil, errors.New("获取租户数据库，设置连接池获取 sql.DB失败:" + err.Error())
	}
	db.SetMaxIdleConns(tenantDataSource.PoolSize)                   // 设置最大空闲连接数
	db.SetMaxOpenConns(tenantDataSource.MaxPoolSize)                // 设置最大连接数
	db.SetConnMaxLifetime(tenantDataSource.ExpMinute * time.Minute) // 设置连接最大生命周期
	return sqlDb, nil
}

// GetDbFromMap 增加数据源到缓存
func GetDbFromMap(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		db := sqlDB.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			_ = dbInfo.Close()
			dbMap.Delete(tenantID)
			//log.Println("移除数据源：", tenantID)
			newDb, err := GetSQLDb(tenantID, crmdb, false)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			//log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := GetSQLDb(tenantID, crmdb, false)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	//log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func GetDbFromMapWithOpt(tenantID string, crmDB *gorm.DB, opt *OptSQL) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			_ = dbInfo.Close()
			dbMap.Delete(tenantID)
			//log.Println("移除数据源：", tenantID)
			newDb, err := getSQLDbWithOpt(tenantID, crmDB, opt, false)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			//log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := getSQLDbWithOpt(tenantID, crmDB, opt, false)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	//log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func senMsgToWx(tenantId string, status sql.DBStats) {
	if status.Idle == 0 {
		go func() {
			errMsg := fmt.Sprintf("租户：%s 数据源出现异常   最大连接：%d,打开连接：%d，使用连接：%d，等待连接：%d", tenantId, status.MaxOpenConnections, status.OpenConnections, status.InUse,
				status.WaitCount)
			log.Println(errMsg)
			_ = utils.SendMsgToWorkWx(utils.DefaultRegChatID, errMsg, utils.WorkWxAPIPath, utils.WorkWxRestTokenStr)
		}()
	}
}

// GetDbFromMapV2 增加数据源到缓存
func GetDbFromMapV2(tenantID string, crmDB *gorm.DB, enable bool) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			_ = dbInfo.Close()
			dbMap.Delete(tenantID)
			//log.Println("移除数据源：", tenantID)
			newDb, err := GetSQLDb(tenantID, crmDB, enable)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			//log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := GetSQLDb(tenantID, crmDB, enable)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	//log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func UpdateMapV2(tenantID string) {
	if sqlDb, isOk := dbMap.Load(tenantID); isOk {
		db := sqlDb.(*gorm.DB)
		sqlDb, _ := db.DB()
		_ = sqlDb.Close()
		dbMap.Delete(tenantID)
		//log.Println("更新数据源：", tenantID)
	}
}
