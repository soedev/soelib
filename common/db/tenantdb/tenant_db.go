package tenantdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/keylock"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/common/utils"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

// dbMap 数据源缓存列表
var dbMap sync.Map

// 健康检查配置
var (
	healthCheckInterval = 30 * time.Second // 健康检查间隔
	healthCheckTimeout  = 5 * time.Second  // 健康检查超时时间
	healthCheckerOnce   sync.Once
	healthCheckerCtx    context.Context
	healthCheckerCancel context.CancelFunc
)

type OptSQL struct {
	DBConfig        gorm.Config
	EnableTrace     bool
	ApplicationName string
}

// 获取租户数据源统一方法：  tenantID（租户编号）、crmDB（获取配置连接源）、optSQL（数据库配置参数）、enable（是否启用备用数据源）
func _getDB(tenantID string, crmDB *gorm.DB, opt *OptSQL, enable bool) (sqlDb *gorm.DB, err error) {
	var tenantDataSource TenantDataSource
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

	if tenantDataSource.DriverClassname == "org.postgresql.ds.PGSimpleDataSource" {
		dbInfo := fmt.Sprintf("host=%s user=%s port=%d dbname=%s sslmode=disable password=%s application_name=%s",
			server, tenantDataSource.UserName, port, dbName, password, opt.ApplicationName)
		sqlDb, err = gorm.Open(postgres.Open(dbInfo), &opt.DBConfig)
	} else {
		dbInfo := fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d;encrypt=disable; application_name=%s", server, tenantDataSource.UserName, password, dbName, port, opt.ApplicationName)
		sqlDb, err = gorm.Open(sqlserver.Open(dbInfo), &opt.DBConfig)
	}

	if err != nil {
		return nil, err
	}
	// 启用链路
	if opt.EnableTrace {
		if err = sqlDb.Use(tracing.NewPlugin()); err != nil {
			return nil, err
		}
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

// GetDbFromMap 获取数据源标准方法 tenantID（租户编号）、crmDB（获取配置连接源）、args（参数列表{程序名称、启用链路、日志级别}）
func GetDbFromMap(tenantID string, crmDB *gorm.DB, args ...interface{}) (*gorm.DB, error) {
	// 启动健康检查器（仅首次调用时启动）
	startHealthChecker()

	// 第一次检查：无锁快速路径，直接从缓存获取
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 缓存未命中，需要创建新连接，使用键锁避免重复创建
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)

	// 第二次检查：可能在等待锁期间已被其他goroutine创建
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 确实需要创建新连接
	applicationName, enableTrace, logLevel := parseArguments(args...)
	newDb, err := _getDB(tenantID, crmDB, &OptSQL{
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logLevel)},
		ApplicationName: applicationName,
		EnableTrace:     enableTrace,
	}, false)

	if err != nil {
		return nil, err
	}

	dbMap.Store(tenantID, newDb)
	soelog.Logger.Info(fmt.Sprintf("创建新数据源连接：租户[%s]", tenantID))
	return newDb, nil
}

// GetDbFromMapWithOpt 根据配置获取数据源 tenantID（租户编号）、crmDB（获取配置连接源）、opt（数据库配置信息）
func GetDbFromMapWithOpt(tenantID string, crmDB *gorm.DB, opt *OptSQL) (*gorm.DB, error) {
	// 启动健康检查器（仅首次调用时启动）
	startHealthChecker()

	// 第一次检查：无锁快速路径，直接从缓存获取
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 缓存未命中，需要创建新连接，使用键锁避免重复创建
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)

	// 第二次检查：可能在等待锁期间已被其他goroutine创建
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 确实需要创建新连接
	newDb, err := _getDB(tenantID, crmDB, opt, false)
	if err != nil {
		return nil, err
	}

	dbMap.Store(tenantID, newDb)
	soelog.Logger.Info(fmt.Sprintf("创建新数据源连接：租户[%s]", tenantID))
	return newDb, nil
}

func senMsgToWx(tenantId string, status sql.DBStats) {
	if status.Idle == 0 {
		go func() {
			errMsg := fmt.Sprintf("租户：%s 数据源出现异常   最大连接：%d,打开连接：%d，使用连接：%d，等待连接：%d", tenantId, status.MaxOpenConnections, status.OpenConnections, status.InUse,
				status.WaitCount)
			log.Println(errMsg)
			utils.SendMsgToWorkWx(utils.DefaultRegChatID, errMsg, utils.WorkWxAPIPath, utils.WorkWxRestTokenStr)
		}()
	}
}

// GetDbFromMapV2 获取数据源扩展方法 tenantID（租户编号）、crmDB（获取配置连接源）、enable（是否启用备库）args（参数列表{程序名称、启用链路、日志级别}）
func GetDbFromMapV2(tenantID string, crmDB *gorm.DB, enable bool, args ...interface{}) (*gorm.DB, error) {
	// 启动健康检查器（仅首次调用时启动）
	startHealthChecker()

	// 第一次检查：无锁快速路径，直接从缓存获取
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 缓存未命中，需要创建新连接，使用键锁避免重复创建
	key := "SQLDB_" + tenantID
	keylock.GetKeyLockIns().Lock(key)
	defer keylock.GetKeyLockIns().Unlock(key)

	// 第二次检查：可能在等待锁期间已被其他goroutine创建
	if sqlDB, isOk := dbMap.Load(tenantID); isOk {
		return sqlDB.(*gorm.DB), nil
	}

	// 确实需要创建新连接
	applicationName, enableTrace, logLevel := parseArguments(args...)
	newDb, err := _getDB(tenantID, crmDB, &OptSQL{
		DBConfig:        gorm.Config{Logger: logger.Default.LogMode(logLevel)},
		ApplicationName: applicationName,
		EnableTrace:     enableTrace,
	}, enable)

	if err != nil {
		return nil, err
	}

	dbMap.Store(tenantID, newDb)
	soelog.Logger.Info(fmt.Sprintf("创建新数据源连接：租户[%s]", tenantID))
	return newDb, nil
}

func parseArguments(args ...interface{}) (string, bool, logger.LogLevel) {
	applicationName := "go-service"
	enableTrace := false
	logLevel := logger.Error // 默认错误级别L
	switch len(args) {
	case 1:
		applicationName = args[0].(string)
	case 2:
		applicationName = args[0].(string)
		enableTrace = args[1].(bool)
	case 3:
		applicationName = args[0].(string)
		enableTrace = args[1].(bool)
		logLevel = args[2].(logger.LogLevel)
	}
	return applicationName, enableTrace, logLevel
}

func UpdateMapV2(tenantID string) {
	if sqlDb, isOk := dbMap.Load(tenantID); isOk {
		db := sqlDb.(*gorm.DB)
		sqlDb, _ := db.DB()
		_ = sqlDb.Close()
		dbMap.Delete(tenantID)
	}
}

// startHealthChecker 启动异步健康检查器（单例模式）
func startHealthChecker() {
	healthCheckerOnce.Do(func() {
		healthCheckerCtx, healthCheckerCancel = context.WithCancel(context.Background())
		go healthCheckLoop(healthCheckerCtx)
		soelog.Logger.Info("租户数据源健康检查器已启动")
	})
}

// StopHealthChecker 停止健康检查器（用于优雅关闭）
func StopHealthChecker() {
	if healthCheckerCancel != nil {
		healthCheckerCancel()
		soelog.Logger.Info("租户数据源健康检查器已停止")
	}
}

// healthCheckLoop 健康检查循环
func healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkAllConnections()
		}
	}
}

// checkAllConnections 检查所有租户连接的健康状态
func checkAllConnections() {
	dbMap.Range(func(key, value interface{}) bool {
		tenantID := key.(string)
		db := value.(*gorm.DB)

		// 使用带超时的context进行Ping检查
		ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
		defer cancel()

		sqlDB, err := db.DB()
		if err != nil {
			soelog.Logger.Error(fmt.Sprintf("租户[%s]获取底层数据库连接失败: %v", tenantID, err))
			removeAndCloseDB(tenantID, db)
			return true
		}

		// 在独立goroutine中执行Ping，避免阻塞其他租户的检查
		done := make(chan error, 1)
		go func() {
			done <- sqlDB.PingContext(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				soelog.Logger.Warn(fmt.Sprintf("租户[%s]连接健康检查失败，将移除缓存: %v", tenantID, err))
				removeAndCloseDB(tenantID, db)
				// 记录连接池状态
				if stats := sqlDB.Stats(); stats.Idle == 0 {
					senMsgToWx(tenantID, stats)
				}
			}
		case <-ctx.Done():
			soelog.Logger.Warn(fmt.Sprintf("租户[%s]连接健康检查超时，将移除缓存", tenantID))
			removeAndCloseDB(tenantID, db)
		}

		return true
	})
}

// removeAndCloseDB 移除并关闭数据库连接
func removeAndCloseDB(tenantID string, db *gorm.DB) {
	dbMap.Delete(tenantID)
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// SetHealthCheckInterval 设置健康检查间隔（用于测试或特殊场景调整）
func SetHealthCheckInterval(interval time.Duration) {
	if interval > 0 {
		healthCheckInterval = interval
		soelog.Logger.Info(fmt.Sprintf("健康检查间隔已设置为: %v", interval))
	}
}

// SetHealthCheckTimeout 设置健康检查超时时间
func SetHealthCheckTimeout(timeout time.Duration) {
	if timeout > 0 {
		healthCheckTimeout = timeout
		soelog.Logger.Info(fmt.Sprintf("健康检查超时时间已设置为: %v", timeout))
	}
}
