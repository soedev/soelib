package client

import (
	"context"
	"fmt"
	"github.com/soedev/soelib/common/soelog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//OpenTLS 是否使用TLS
var (
	openTLS  = false
	RPCConn  *grpc.ClientConn
	metadata = map[string]string{}
)

// customCredential 自定义认证
type customCredential struct{}

type GrpcConfig struct {
	Host    string //服务IP
	Port    string //服务端口
	OpenTLS bool   //是否使用 tls
	KeyPath string //密码
}

// GetRequestMetadata 实现自定义认证接口
func (c customCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return metadata, nil
}

// RequireTransportSecurity 自定义认证是否开启TLS
func (c customCredential) RequireTransportSecurity() bool {
	return openTLS
}

//InitRPC 初始化 RPC 客户端
func InitRPC(config GrpcConfig, aMetadata map[string]string) (err error) {
	openTLS = config.OpenTLS
	metadata = aMetadata
	var opts []grpc.DialOption
	if openTLS {
		// TLS连接  sync  这个name很重要，一定要和生成时的Common Name 要对应上
		creds, err := credentials.NewClientTLSFromFile(config.KeyPath, "sync")
		if err != nil {
			soelog.Logger.Fatal("加载证书失败 " + err.Error())
			return err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	// 使用自定义认证
	opts = append(opts, grpc.WithPerRPCCredentials(new(customCredential)))
	RPCConn, err = grpc.Dial(fmt.Sprintf("%s:%s", config.Host, config.Port), opts...) //这里默认连接的是本地
	if err != nil {                                                                   //这里好像检测不到是否能够连接到服务器
		soelog.Logger.Fatal("连接不到远程grpc服务器: " + err.Error())
	}
	return nil
}

//CloseRPC 关闭PRC连接
func CloseRPC() {
	RPCConn.Close()
}

//ResetRPC 重置RPC连接
func ResetRPC(config GrpcConfig, aMetadata map[string]string) {
	CloseRPC()
	InitRPC(config, aMetadata)
}
