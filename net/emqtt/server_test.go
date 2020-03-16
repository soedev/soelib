package emqtt

import "testing"

//服务端测试
func TestServer(t *testing.T) {
	StartServer(ServerConfig{
		Port:    "1884",
		WsAddr:  "18080",
		WssAddr: "18081",
	})
}
