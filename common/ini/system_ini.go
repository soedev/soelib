package ini

/**
  system.ini 文件读取处理类
*/

import (
	"github.com/go-ini/ini"
	"github.com/soedev/soelib/common/des"
	"log"
	"strconv"
	"strings"
)

type Database struct {
	Server   string
	DbName   string
	User     string
	Password string
	Port     int
}

//TCPSetting 小索通信配置
type TCPConfig struct {
	Host string
	Port string
}

//MQTT 通讯配置类
type MqttConfig struct {
	ThirdServerIP   string
	ThirdServerPort string
	ServerPort      string
	WsAddr          string
	WssAddr         string
}

//MqttSetting MQTT消息队列配置
var MqttSetting = &MqttConfig{}

//TCPSetting 小索通信配置
var TCPSetting = &TCPConfig{}

//DatabaseSetting 数据库配置
var DatabaseSetting = &Database{}

//LoadSystemIni 加载ini 文件
func LoadSystemIni(iniFile string) {
	var err error
	cfg, err := ini.Load(iniFile)
	if err != nil {
		log.Println(".ini文件出错:" + err.Error())
		return
	}
	//读取小索辅助配置信息
	TCPSetting.Host = cfg.Section("TCP").Key("Host").String()
	TCPSetting.Port = cfg.Section("TCP").Key("Port").String()
	if TCPSetting.Host == "" {
		TCPSetting.Host = "127.0.0.1"
	}
	if TCPSetting.Port == "" {
		TCPSetting.Port = "5201"
	}

	//读取 MQTT通讯配置
	MqttSetting.ServerPort = cfg.Section("MQTT").Key("ServerPort").String()
	if MqttSetting.ServerPort == "" {
		MqttSetting.ServerPort = "2883"
	}
	MqttSetting.WsAddr = cfg.Section("MQTT").Key("WsAddr").String()
	MqttSetting.WssAddr = cfg.Section("MQTT").Key("WssAddr").String()
	if MqttSetting.WsAddr == "" {
		MqttSetting.WsAddr = "18080"
	}
	if MqttSetting.WssAddr == "" {
		MqttSetting.WssAddr = "18081"
	}
	innDb := cfg.Section("DataBase").Key("File").String()
	DatabaseSetting.DbName, DatabaseSetting.Server, DatabaseSetting.Port = parseDbFile(innDb)
	DatabaseSetting.User = cfg.Section("DataBase").Key("UserName").String()
	DatabaseSetting.Password = cfg.Section("DataBase").Key("Password").String()

	//处理数据库链接 加密信息
	powerDes := des.PowerDes{}
	if DatabaseSetting.User == "" {
		DatabaseSetting.User = "sa"
	} else {
		decryStr, err := powerDes.PowerDecryStr(DatabaseSetting.User, des.PowerDesKey)
		if err == nil {
			DatabaseSetting.User = decryStr
		}
	}
	if DatabaseSetting.Password == "" {
		DatabaseSetting.User = "soesoft"
	} else {
		decryStr, err := powerDes.PowerDecryStr(DatabaseSetting.Password, des.PowerDesKey)
		if err == nil {
			DatabaseSetting.Password = decryStr
		}
	}
}

func parseDbFile(fileValue string) (dbname string, server string, port int) {
	server = strings.Split(fileValue, ":")[0]
	dbname = strings.Split(fileValue, ":")[1]
	a := strings.Index(server, ",")
	if a < 0 {
		port = 1433
	} else {
		port, _ = strconv.Atoi(server[a+1:])
		server = server[:a]
	}
	return dbname, server, port
}
