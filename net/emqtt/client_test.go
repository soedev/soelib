package emqtt

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"sync"
	"testing"
)

var wg sync.WaitGroup

func TestClient(t *testing.T) {
	//初始化订阅信息： 不需要订阅可以传空数组
	ts := []Topic{
		{
			Name: "topic1",  //订阅通道
			Fun:  emqttCall, //回调函数
		}, {
			Name: "topic2",  //订阅通道
			Fun:  emqttCall, //回调函数
		},
	}
	config := ClientConfig{
		Server:   "192.168.1.129",
		Port:     "1883",
		UserName: "",
		Password: "",
	}
	//订阅通道
	if OpenClientUseAuth("testPrefix", config, ts) {
		wg.Add(1)
		defer func() {
			Client.Unsubscribe("") //取消关闭全部
			Client.Close()         //关闭连接
		}()
	}
	wg.Wait()
	fmt.Println("退出")
}

//通道回调函数
var emqttCall mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	wg.Done()
	fmt.Println(fmt.Sprintf("收到EMQTT消息 通道[%s]  消息内容[%s]", msg.Topic(), msg.Payload()))
}
