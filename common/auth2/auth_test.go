package auth2

import (
	"fmt"
	"github.com/soedev/soelib/net/grpc/client"
	pb "github.com/soedev/soelib/net/grpc/proto"
	"testing"
)

func TestAuth(t *testing.T) {
	conf := AuthTokenConfig{
		Grpc: client.GrpcConfig{
			Host:    "127.0.0.1",
			Port:    "8090",
			KeyPath: "",
			OpenTLS: false,
		},
		AccessType: "rest",
		RestUrl:    "http://127.0.0.1:8099",
	}
	isR, err := InitService(conf, map[string]string{
		"appid":  "101010",
		"appkey": "i am key1",
	})
	if err != nil {
		fmt.Println("初始化 auth service " + err.Error())
	}
	if isR {
		defer Release()
	}
	var msg = &pb.AuthResponse{
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJ3ZWJfc2Fhc19jYXNoIiwiaXNzIjoiNjU1NSIsImp0aSI6IjlhMWYwYWY5NDFhNDRkZjdiZWM1NDhlMjk0M2VkN2IwIiwic3ViIjoic29lNWNjNDI5MzY0NmUwZmIwMDAxYjIzN2QxIn0.VAyP2OIr2C4qY9wb5FCEkgzE0q_Y5Z40ItKCi0v8B64",
	}
	tables, err := AuthClient.Service.AuthToken(msg)
	if err != nil {
		fmt.Print(fmt.Sprintf(" 调用验证服务发生错误：%s", err.Error()))
		return
	}
	fmt.Println(" 调用服务成功")
	fmt.Println(tables.Message)
}
