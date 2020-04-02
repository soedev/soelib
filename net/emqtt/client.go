package emqtt

/**
  client   EMQTT 客户端  连接远端EMQTT服务
*/
import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"strconv"
	"time"
)

//订阅通道信息
type Topic struct {
	Name string              //订阅名称
	Fun  mqtt.MessageHandler //订阅通道
}

//EmqttClient 客户端
type EmqttClient struct {
	Server string
	Port   string
	MQTT   mqtt.Client //EMQTT 连接
	Topics []Topic     //订阅信息
}

type ClientConfig struct {
	Server   string //服务IP
	Port     string //服务端口
	UserName string //用户名 【如果需要验证】
	Password string //密码
}

var Client *EmqttClient

func OpenClientUseAuth(clientPrefix string, config ClientConfig, subTopics []Topic) bool {
	timeUnixNano := time.Now().UnixNano()
	tcpStr := fmt.Sprintf("tcp://%s:%s", config.Server, config.Port)
	opts := mqtt.NewClientOptions().AddBroker(tcpStr).SetClientID(clientPrefix + strconv.FormatInt(timeUnixNano, 10))
	opts.SetUsername(config.UserName)
	opts.SetPassword(config.Password)
	//opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(f)
	//opts.SetPingTimeout(1 * time.Second)
	opts.OnConnect = onClientBack
	Client = &EmqttClient{
		Server: config.Server,
		Port:   config.Port,
		MQTT:   nil,
		Topics: subTopics,
	}
	if token := mqtt.NewClient(opts).Connect(); token.Wait() && token.Error() != nil {
		log.Println(fmt.Sprintf("连接 mqtt:%s 发生错误 %s", tcpStr, token.Error().Error()))
		return false
	}
	return true
}

//连接成功 回调
func onClientBack(client mqtt.Client) {
	Client.MQTT = client
	log.Println(fmt.Sprintf("连接上了 mqtt:tcp://%s:%s", Client.Server, Client.Port))
	for i := 0; i < len(Client.Topics); i++ {
		Client.Subscribe(Client.Topics[i].Name, Client.Topics[i].Fun)
	}
}

func (s *EmqttClient) Close() {
	if s.MQTT == nil {
		return
	}
	if s.MQTT.IsConnected() {
		s.MQTT.Disconnect(250) //断开连接
	}
}

func (s *EmqttClient) Subscribe(topic string, callback mqtt.MessageHandler) {
	if s.MQTT == nil {
		log.Println("EMQTT 未连接上服务器")
		return
	}
	if s.MQTT.IsConnected() {
		token := s.MQTT.Subscribe(topic, 0, callback)
		if token.Wait() && token.Error() != nil {
			log.Println(fmt.Sprintf("订阅EMQTT通道%s 发生错误:%s", topic, token.Error().Error()))
		} else {
			log.Println(fmt.Sprintf("订阅EMQTT通道%s 成功", topic))
		}
	} else {
		log.Println("EMQTT 未连接上服务器")
		return
	}
}

//取消订阅
func (s *EmqttClient) Unsubscribe(topic string) {
	if s.MQTT == nil {
		log.Println("EMQTT 未连接上服务器")
		return
	}
	for i := 0; i < len(Client.Topics); i++ {
		if topic != "" {
			if Client.Topics[i].Name != topic {
				continue
			}
		}
		if token := s.MQTT.Unsubscribe(Client.Topics[i].Name); token.Wait() && token.Error() != nil {
			log.Println(fmt.Sprintf("取消订阅EMQTT通道%s 发生错误:%s", Client.Topics[i].Name, token.Error().Error()))
		} else {
			log.Println(fmt.Sprintf("取消订阅EMQTT通道%s 成功", Client.Topics[i].Name))
		}
	}
}

//发送消息到通道
func (s *EmqttClient) Publish(topic string, msg []byte) {
	if s.MQTT == nil {
		log.Println("EMQTT 未连接上服务器")
		return
	}
	if s.MQTT.IsConnected() {
		s.MQTT.Publish(topic, 0, false, string(msg))
	} else {
		log.Println(fmt.Sprintf("发生消息到通道%s错误，Emqtt 未连接", topic))
	}
}

func (s *EmqttClient) PublishStrMsg(topic string, msg string) {
	if s.MQTT == nil {
		log.Println("EMQTT 未连接上服务器")
		return
	}
	if s.MQTT.IsConnected() {
		log.Println("发送消息："+msg)
		s.MQTT.Publish(topic, 0, false, msg)
	} else {
		log.Println(fmt.Sprintf("发生消息到通道%s错误，Emqtt 未连接", topic))
	}
}

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("TOPIC: %s\n", msg.Topic())
	log.Printf("MSG: %s\n", msg.Payload())
}
