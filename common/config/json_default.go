package config

/**
  默认的读取json 配置信息  ./config.json
  如果您的项目配置格式特殊  请自行添加自己的配置文件，在自己的项目中
  本配置 是公共连接配置 不需要任何代码添加
*/

import (
	"encoding/json"
	"errors"
	"github.com/soedev/soelib/common/auth2"
	"github.com/soedev/soelib/common/db/specialdb"
	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/common/soesentry"
	"github.com/soedev/soelib/net/emqtt"
	"github.com/soedev/soelib/net/soehttp"
	"github.com/soedev/soelib/net/soetcp"
	"github.com/soedev/soelib/net/soetrace"
	"github.com/soedev/soelib/tools/nacos"
	"os"
)

const (
	RabbitConfigDataID  = "rabbit.config"
	RabbitConfigGroupID = "soe"
)

type JsonConfig struct {
	MongoConfig   specialdb.MongoConfig //mogo 数据库连接配置
	RedisConfig   specialdb.RedisConfig //redis 连接配置
	TraceConfig   soetrace.OtelTracerConfig
	HTTPConfig    soehttp.SoeHTTPConfig
	TCP           soetcp.TcpConfig //小索辅助配置
	MQTT          emqttConfig      //MQTT通讯配置
	ATT           attConfig        //中控考勤机 bs 模式处理配置信息
	Caller        callerConfig     //来电显示盒配置
	AuthToken     auth2.AuthTokenConfig
	Sentry        soesentry.Sentry
	Kafka         kafka
	SlowInterface slowInterface
	Rabbit        Rabbit
	AliRabbit     Rabbit
	AcmConfig     nacos.AcmConfig
}

// emqtt  服务端以及客户端配置
type emqttConfig struct {
	Client emqtt.ClientConfig
	Server emqtt.ServerConfig
}

// 考勤机配置
type attConfig struct {
	Delay      int //延迟
	ErrorDelay int
	TimeZone   int
	Realtime   int
}

// 来电显示盒子配置
type callerConfig struct {
	LineCount int  //来电路数
	Enable    bool //是否开启来电显示
}

type kafka struct {
	Server string
}

type slowInterface struct {
	SlowTime int
	Tag      string
}
type Rabbit struct {
	Host     string
	Port     int
	Username string
	Password string
	Vhost    string
}
type RabbitConsumerInfo struct {
	//交换机
	ExchangeName string
	//队列
	QueueName string
	//模式
	ExchangeType string
}

// JsonConfig 配置信息
var Config JsonConfig

// 加载配置文件
func LoadConfig(configFile string) {
	file, err := os.Open(configFile)
	if err != nil {
		soelog.Logger.Fatal("读取config配置文件,发生错误:" + err.Error())
	}
	decoder := json.NewDecoder(file)
	Config = JsonConfig{}
	err = decoder.Decode(&Config)
	if err != nil {
		soelog.Logger.Fatal("config配置文件转换错误,请检查文件格式是否正确 错误信息:" + err.Error())
	}
	Config.Check()
}

// 配置默认值检测
func (s *JsonConfig) Check() {
	if s.TCP.Host == "" {
		s.TCP.Host = "127.0.0.1"
	}
	if s.TCP.Port == "" {
		s.TCP.Port = "5201"
	}

	//设置心跳延迟
	if s.ATT.Delay <= 0 {
		s.ATT.Delay = 5
	}
	if s.ATT.ErrorDelay <= 0 {
		s.ATT.ErrorDelay = 10
	}
	if s.ATT.TimeZone <= 0 {
		s.ATT.TimeZone = 8
	}
	if s.ATT.Realtime <= 0 {
		s.ATT.Realtime = 1
	}
	if s.Caller.LineCount == 0 {
		s.Caller.LineCount = 1
	}

	//读取 MQTT通讯配置
	if s.MQTT.Server.Port == "" {
		s.MQTT.Server.Port = "1883"
	}
	if s.MQTT.Server.WsAddr == "" {
		s.MQTT.Server.WsAddr = "18080"
	}
	if s.MQTT.Server.WssAddr == "" {
		s.MQTT.Server.WssAddr = "18081"
	}

	if s.MongoConfig.PoolLimit <= 0 || s.MongoConfig.PoolLimit > 4096 {
		s.MongoConfig.PoolLimit = 100
	}

	//全链路跟踪默认值配置： 采样率检测
	if s.TraceConfig.SamplingRatio <= 0 || s.TraceConfig.SamplingRatio > 1 {
		s.TraceConfig.SamplingRatio = 0.1
	}

	//HTTP 默认值配置 熔断默认配置
	if s.HTTPConfig.Hystrix.Timeout == 0 {
		s.HTTPConfig.Hystrix.Timeout = 5000 //执行command的超时时间(毫秒)
	}
	if s.HTTPConfig.Hystrix.MaxConcurrentRequests == 0 {
		s.HTTPConfig.Hystrix.MaxConcurrentRequests = 8 //command的最大并发量
	}
	if s.HTTPConfig.Hystrix.SleepWindow == 0 {
		s.HTTPConfig.Hystrix.SleepWindow = 1000 //过多长时间，熔断器再次检测是否开启。单位毫秒
	}
	if s.HTTPConfig.Hystrix.ErrorPercentThreshold == 0 {
		s.HTTPConfig.Hystrix.ErrorPercentThreshold = 30 //错误率 请求数量大于等于RequestVolumeThreshold并且错误率到达这个百分比后就会启动
	}
	if s.HTTPConfig.Hystrix.RequestVolumeThreshold == 0 {
		s.HTTPConfig.Hystrix.RequestVolumeThreshold = 5 //请求阈值(一个统计窗口10秒内请求数量)  熔断器是否打开首先要满足这个条件；这里的设置表示至少有5个请求才进行ErrorPercentThreshold错误百分比计算
	}
	//HTTP 默认值配置 告警
	if s.HTTPConfig.Alarm.ApiPath == "" {
		s.HTTPConfig.Alarm.ApiPath = "https://www.soesoft.org/workwx-rest/api/send-msg-to-chat"
	}

	//AuthToken 配置
	if s.AuthToken.AccessType == "" {
		s.AuthToken.AccessType = "grpc"
	}
	if s.AuthToken.Grpc.Port == "" {
		s.AuthToken.Grpc.Port = "8090"
	}
	if s.AuthToken.Grpc.Host == "" {
		s.AuthToken.Grpc.Host = "127.0.0.1"
	}
	if s.AcmConfig.AccessKey != "" {
		s.AcmConfig.AccessKey = des.DecryptDESECB([]byte(s.AcmConfig.AccessKey), des.DesKey)
	}
	if s.AcmConfig.SecretKey != "" {
		s.AcmConfig.SecretKey = des.DecryptDESECB([]byte(s.AcmConfig.SecretKey), des.DesKey)
	}
}

// GetAcmConfig 获取acm相关连接
func GetAcmConfig(dataID string, groupID string, config *JsonConfig) error {
	if nacos.AcmClient == nil {
		return errors.New("acm连接失败")
	}
	content, err := nacos.GetAcmContent(dataID, groupID)
	if err != nil {
		soelog.Logger.Error("获取acm内容错误：" + err.Error())
		return err
	} else if content != "" && content != "{}" {
		switch dataID {
		case RabbitConfigDataID:
			json.Unmarshal([]byte(content), &config.Rabbit)
			return nil
		}
	}
	return nil
}
