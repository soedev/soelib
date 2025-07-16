package specialdb

import (
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
	"regexp"
	"time"
)

type DbConfig struct {
	DBInfo          string
	Dialect         string
	DBConfig        gorm.Config
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	EnableTrace     bool
}

// ConnDB 根据索易配置信息连接数据库
//
//	DBConfig：目前支持以下配置：
//	&gorm.Config{
//	  Logger: logger.Default.LogMode(logger.Info), // 设置日志级别为 Info 默认为 default 只打印错误日志
//	  NamingStrategy: schema.NamingStrategy{ 默认未设置
//	  TablePrefix:   "t_",  // 表名前缀
//	  SingularTable: true, // 单数表名
//	  },
//	  SkipDefaultTransaction: true, // 跳过默认事务 默认false
//	  PrepareStmt:            true, // 启用预编译 默认false
//	  QueryFields:            true, // 查询时指定字段 默认false
//	}
func ConnDB(config DbConfig) (*gorm.DB, error) {
	// 1. 判断数据库类型
	var db *gorm.DB
	var err error
	var dialect gorm.Dialector
	if config.Dialect == "mysql" {
		dialect = mysql.Open(config.DBInfo)
	} else if config.Dialect == "postgres" {
		dialect = postgres.Open(config.DBInfo)
	} else if config.Dialect == "sqlserver" {
		dialect = sqlserver.Open(config.DBInfo)
	} else {
		return nil, errors.New("不支持的数据库类型")
	}
	// 2. 连接数据库
	db, err = gorm.Open(dialect, &config.DBConfig)
	if err != nil {
		return nil, errors.New("数据库连接失败:" + err.Error())
	}

	// 启用链路追踪插件
	if config.EnableTrace {
		if err = db.Use(tracing.NewPlugin()); err != nil {
			return nil, errors.New("启用链路失败:" + err.Error())
		}
	}

	// 3. 设置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.New("设置连接池：获取 sql.DB失败:" + err.Error())
	}
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)       // 设置最大空闲连接数
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)       // 设置最大连接数
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime) // 设置连接最大生命周期

	return db, nil
}

// Transaction 统一事务
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

// FilteredSQLInject 正则过滤sql注入的方法
// 参数 : 要匹配的语句
func FilteredSQLInject(toMatchStr string) bool {
	//过滤 ‘
	//ORACLE 注解 --  /**/
	//关键字过滤 update ,delete
	// 正则的字符串, 不能用 " " 因为" "里面的内容会转义
	str := `(?:')|(?:--)|(/\\*(?:.|[\\n\\r])*?\\*/)|(\b(select|update|and|or|delete|insert|trancate|char|chr|into|substr|ascii|declare|exec|count|master|into|drop|execute)\b)`
	re, err := regexp.Compile(str)
	if err != nil {
		return false
	}
	return re.MatchString(toMatchStr)
}
