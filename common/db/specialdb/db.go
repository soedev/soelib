package specialdb

import (
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"regexp"
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

// Setup 初始化数据库
func ConnDB(config DbConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	if config.Dialect == "mysql" {
		dialect := mysql.New(mysql.Config{
			DSN: config.DBInfo},
		)
		db, err = gorm.Open(dialect)
		if err != nil {
			return nil, err
		}
	} else if config.Dialect == "postgres" {
		dialect := postgres.Open(config.DBInfo)
		db, err = gorm.Open(dialect)
		if err != nil {
			return nil, err
		}
	}
	Db, _ := db.DB()
	Db.SetMaxIdleConns(config.MaxIdleConns)       //最大空闲数
	Db.SetMaxOpenConns(config.MaxOpenConns)       //最大连接数
	Db.SetConnMaxLifetime(config.ConnMaxLifetime) //设置最大空闲时间，超过将关闭连接
	return db, nil
}

// Transaction 统一事务
func Transaction(tx *gorm.DB, fn func() error) (err error) {
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

// 正则过滤sql注入的方法
// 参数 : 要匹配的语句
func FilteredSQLInject(to_match_str string) bool {
	//过滤 ‘
	//ORACLE 注解 --  /**/
	//关键字过滤 update ,delete
	// 正则的字符串, 不能用 " " 因为" "里面的内容会转义
	str := `(?:')|(?:--)|(/\\*(?:.|[\\n\\r])*?\\*/)|(\b(select|update|and|or|delete|insert|trancate|char|chr|into|substr|ascii|declare|exec|count|master|into|drop|execute)\b)`
	re, err := regexp.Compile(str)
	if err != nil {
		panic(err.Error())
		return false
	}
	return re.MatchString(to_match_str)
}
