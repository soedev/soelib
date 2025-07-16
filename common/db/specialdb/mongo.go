package specialdb

import (
	"fmt"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

type MongoConfig struct {
	DSN         string
	PoolLimit   int
	DbName      string
	EnableTrace bool
}

// ConnMongo 初始化数据库
func ConnMongo(config MongoConfig) (*mongo.Client, error) {
	opts := options.Client()
	mps := uint64(config.PoolLimit)
	if config.EnableTrace {
		opts.Monitor = otelmongo.NewMonitor()
	}
	opts.MaxPoolSize = &mps
	opts.ApplyURI(config.DSN)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败:%s", err.Error())
	}
	return client, nil
}
