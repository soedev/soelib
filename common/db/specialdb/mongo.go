package specialdb

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"time"
)

type MongoConfig struct {
	Host     string
	Port     uint
	UserName string
	Password string
	DbName   string
}

//MogSession 数据库连接
var MongoSession *mgo.Session

//SetupMog 初始化数据库
func ConnMongo(config MongoConfig) error {
	//连接mongo
	mongoDbInfo := fmt.Sprintf("mongodb://%s:%s@%s:%d", config.UserName, config.Password, config.Host, config.Port)
	if config.Host != "192.168.1.88" {
		mongoDbInfo = config.Host
	}
	session, err := mgo.Dial(mongoDbInfo)
	if err != nil {
		fmt.Println(err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	session.SetPoolLimit(2000) //默认4096个连接
	session.SetSocketTimeout(time.Second * 5)
	MongoSession = session
	return nil
}
