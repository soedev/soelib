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
	"log"
	"sync"
	"time"
)

// dbMap 数据源缓存列表
var dbMap sync.Map

type OptSQL struct {
	LogMode         bool
	ApplicationName string
}

func (o *OptSQL) config() {
	if o.ApplicationName == "" {
		o.ApplicationName = "go-service"
	}
}

func GetSQLDb(tenantID string, crmdb *gorm.DB, enable bool) (*gorm.DB, error) {
	opt := &OptSQL{
		LogMode:         false,
		ApplicationName: "go-saas-service",
	}
	return getSQLDbWithOpt(tenantID, crmdb, opt, enable)
}

func getSQLDbWithOpt(tenantID string, crmdb *gorm.DB, opt *OptSQL, enable bool) (*gorm.DB, error) {
	var tenantDataSource TenantDataSource
	var err error
	isNew := enable
	if enable {
		tenantDS := TenantDataSourceBack{Db: crmdb}
		tenantDataSourceBack, err := tenantDS.GetByTenantID(tenantID)
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
		tenantDS := TenantDataSource{Db: crmdb}
		tenantDataSource, err = tenantDS.GetByTenantID(tenantID)
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
	sqlDb, err := gorm.Open(sqlserver.Open(dbInfo), &gorm.Config{})
	if tenantDataSource.DriverClassname == "org.postgresql.ds.PGSimpleDataSource" {
		dbInfo = fmt.Sprintf("host=%s user=%s port=%d dbname=%s sslmode=disable password=%s application_name=%s",
			server, tenantDataSource.UserName, port, dbName, password, opt.ApplicationName)
		sqlDb, err = gorm.Open(postgres.Open(dbInfo), &gorm.Config{})
	}
	if err != nil {
		return nil, err
	}
	opt.config()
	if tenantDataSource.MaxPoolSize == 0 {
		tenantDataSource.MaxPoolSize = 10
	}
	if tenantDataSource.PoolSize == 0 {
		tenantDataSource.PoolSize = 2
	}
	if tenantDataSource.ExpMinute == 0 {
		tenantDataSource.ExpMinute = 5
	}

	db, _ := sqlDb.DB()
	db.SetMaxIdleConns(tenantDataSource.PoolSize)                   //最大空闲数
	db.SetMaxOpenConns(tenantDataSource.MaxPoolSize)                //最大连接数
	db.SetConnMaxLifetime(tenantDataSource.ExpMinute * time.Minute) //设置最大空闲时间，超过将关闭连接
	//sqlDb.Logger = logger.Interface.LogMode(logger)
	//sqlDb.LogMode(opt.LogMode)
	return sqlDb, nil
}

// GetDbFromMap 增加数据源到缓存
func GetDbFromMap(tenantID string, crmdb *gorm.DB) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			dbInfo.Close()
			dbMap.Delete(tenantID)
			log.Println("移除数据源：", tenantID)
			newDb, err := GetSQLDb(tenantID, crmdb, false)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := GetSQLDb(tenantID, crmdb, false)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func GetDbFromMapWithOpt(tenantID string, crmdb *gorm.DB, opt *OptSQL) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			dbInfo.Close()
			dbMap.Delete(tenantID)
			log.Println("移除数据源：", tenantID)
			newDb, err := getSQLDbWithOpt(tenantID, crmdb, opt, false)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := getSQLDbWithOpt(tenantID, crmdb, opt, false)
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

// GetDbFromMapV2 增加数据源到缓存
func GetDbFromMapV2(tenantID string, crmdb *gorm.DB, enable bool) (*gorm.DB, error) {
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)
	if sqldb, isOk := dbMap.Load(tenantID); isOk {
		db := sqldb.(*gorm.DB)
		dbInfo, _ := db.DB()
		//go senMsgToWx(tenantID, db.DB().Stats())
		if err := dbInfo.Ping(); err != nil {
			dbInfo.Close()
			dbMap.Delete(tenantID)
			log.Println("移除数据源：", tenantID)
			newDb, err := GetSQLDb(tenantID, crmdb, enable)
			if err != nil {
				return nil, err
			}
			dbMap.Store(tenantID, newDb)
			log.Println(fmt.Sprintf("增加数据源：%s", tenantID))
			return newDb, nil
		}
		return db, nil
	}
	newDb, err := GetSQLDb(tenantID, crmdb, enable)
	if err != nil {
		return nil, err
	}
	dbMap.Store(tenantID, newDb)
	log.Println("增加数据源：", tenantID)
	return newDb, nil
}

func UpdateMapV2(tenantID string) {
	if sqlDb, isOk := dbMap.Load(tenantID); isOk {
		db := sqlDb.(*gorm.DB)
		sqlDb, _ := db.DB()
		sqlDb.Close()
		dbMap.Delete(tenantID)
		log.Println("更新数据源：", tenantID)
	}
}
