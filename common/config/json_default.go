package config

/**
  默认的读取json 配置信息  ./config.json
  如果您的项目配置格式特殊  请自行添加自己的配置文件，在自己的项目中
  本配置 是公共连接配置 不需要任何代码添加
*/

import (
	"encoding/json"
	"github.com/soedev/soelib/common/db/specialdb"
	"github.com/soedev/soelib/common/des"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/net/emqtt"
	"github.com/soedev/soelib/net/soetcp"
	"github.com/soedev/soelib/net/soetrace"
	"os"
)

type JsonConfig struct {
	MongoConfig specialdb.MongoConfig //mogo 数据库连接配置
	RedisConfig specialdb.RedisConfig //redis 连接配置
	TraceConfig soetrace.JaegerTracerConfig
	TCP         soetcp.TcpConfig //小索辅助配置
	MQTT        emqttConfig      //MQTT通讯配置
	ATT         attConfig        //中控考勤机 bs 模式处理配置信息
	Caller      callerConfig     //来电显示盒配置
}

//emqtt  服务端以及客户端配置
type emqttConfig struct {
	Client emqtt.ClientConfig
	Server emqtt.ServerConfig
}

//考勤机配置
type attConfig struct {
	Delay      int //延迟
	ErrorDelay int
	TimeZone   int
	Realtime   int
}

//来电显示盒子配置
type callerConfig struct {
	LineCount int  //来电路数
	Enable    bool //是否开启来电显示
}

//JsonConfig 配置信息
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

//配置默认值检测
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

	//密码密文信息处理
	if s.RedisConfig.Password != "" {
		s.RedisConfig.Password = des.DecryptDESECB([]byte(s.RedisConfig.Password), des.DesKey)
	}
	if s.MongoConfig.Password != "" {
		s.MongoConfig.Password = des.DecryptDESECB([]byte(s.MongoConfig.Password), des.DesKey)
	}

	//全链路跟踪默认值配置
	if s.TraceConfig.Config.Sampler.Type == "" {
		s.TraceConfig.Config.Sampler.Type = "const" //固定采样
	}
	if s.TraceConfig.Config.Sampler.Param == 0 {
		s.TraceConfig.Config.Sampler.Param = 1 //全采样
	}
}
