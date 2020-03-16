package emqtt

/**
  server   EMQTT 服务端 开启服务管理
*/

import (
	"errors"
	"flag"
	"fmt"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/tools/surgemq/message"
	"github.com/soedev/soelib/tools/surgemq/service"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
)

//EmqttServer 服务端
type EmqttServer struct {
	MQTT *service.Server
}

type ServerConfig struct {
	Port    string //开放访问的端口
	WsAddr  string //开放的web 端口
	WssAddr string //开放的web https 端口
}

//Server Mqtt服务端
var Server *EmqttServer

//StartServer 启动 EMQTT服务端
func StartServer(config ServerConfig) {
	var (
		keepAlive      int
		connectTimeout int
		ackTimeout     int
		timeoutRetries int
		//authenticator    string
		sessionsProvider string
		topicsProvider   string
		cpuprofile       string
		wssCertPath      string // path to HTTPS public key
		wssKeyPath       string // path to HTTPS private key
	)

	flag.IntVar(&keepAlive, "keepalive", service.DefaultKeepAlive, "Keepalive (sec)")
	flag.IntVar(&connectTimeout, "connecttimeout", service.DefaultConnectTimeout, "Connect Timeout (sec)")
	flag.IntVar(&ackTimeout, "acktimeout", service.DefaultAckTimeout, "Ack Timeout (sec)")
	flag.IntVar(&timeoutRetries, "retries", service.DefaultTimeoutRetries, "Timeout Retries")
	//flag.StringVar(&authenticator, "auth", service.DefaultAuthenticator, "Authenticator Type")
	flag.StringVar(&sessionsProvider, "sessions", service.DefaultSessionsProvider, "Session Provider Type")
	flag.StringVar(&topicsProvider, "topics", service.DefaultTopicsProvider, "Topics Provider Type")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "CPU Profile Filename")
	flag.StringVar(&wssCertPath, "wsscertpath", "", "HTTPS hdwms-server public key file")
	flag.StringVar(&wssKeyPath, "wsskeypath", "", "HTTPS hdwms-server private key file")

	Server = &EmqttServer{
		MQTT: &service.Server{
			KeepAlive:        keepAlive,
			ConnectTimeout:   connectTimeout,
			AckTimeout:       ackTimeout,
			TimeoutRetries:   timeoutRetries,
			SessionsProvider: sessionsProvider,
			TopicsProvider:   topicsProvider,
		},
	}
	var f *os.File
	var err error
	if cpuprofile != "" {
		f, err = os.Create(cpuprofile)
		if err != nil {
			soelog.Logger.Fatal(err.Error())
		}
		pprof.StartCPUProfile(f)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigchan
		soelog.Logger.Error("Existing due to trapped signal;" + sig.String())

		if f != nil {
			soelog.Logger.Error("Stopping profile")
			pprof.StopCPUProfile()
			f.Close()
		}

		Server.MQTT.Close()

		os.Exit(0)
	}()

	mqttAddr := `tcp://:` + config.Port
	if config.WsAddr != "" || config.WssAddr != "" {
		addr := `tcp://127.0.0.1:` + config.Port
		if err := AddWebsocketHandler("/mqtt", addr); err != nil {
			log.Println(fmt.Sprintf(`Websocket Handler error: %s`, err.Error()))
		}
		/* start a plain websocket listener */
		if config.WsAddr != "" {
			go ListenAndServeWebsocket(":" + config.WsAddr)
			log.Println(fmt.Sprintf(`开启MQTT WS 访问端口: %s`, config.WsAddr))
		}
		/* start a secure websocket listener */
		if config.WssAddr != "" && len(wssCertPath) > 0 && len(wssKeyPath) > 0 {
			go ListenAndServeWebsocketSecure(":"+config.WssAddr, wssCertPath, wssKeyPath)
			log.Println(fmt.Sprintf(`开启MQTT WSS 访问端口: %s`, config.WssAddr))
		}
	}
	log.Println(fmt.Sprintf(`开启MQTT服务成功: %s`, config.Port))
	defer Server.MQTT.Close()
	err = Server.MQTT.ListenAndServe(mqttAddr)
	if err != nil {
		log.Println(fmt.Sprintf(`开启MQTT服务失败: %s`, err.Error()))
	}
}

func (s *EmqttServer) GetClients() ([]service.Svc, error) {
	if s.MQTT == nil {
		return []service.Svc{}, errors.New("broker is close")
	}
	return s.MQTT.GetClients(), nil
}

//Publish 发送信息
func (s *EmqttServer) Publish(topic string, msg []byte) {
	if s.MQTT != nil {
		publicMsg := message.NewPublishMessage()
		publicMsg.SetTopic([]byte(topic))
		publicMsg.SetQoS(0)
		publicMsg.SetPayload(msg)
		s.MQTT.Publish(publicMsg, nil)
	}
}
