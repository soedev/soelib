package auth2

import (
	"github.com/soedev/soelib/net/grpc/client"
	pb "github.com/soedev/soelib/net/grpc/proto"
)

type AuthTokenConfig struct {
	AccessType string            //来电路数
	RestUrl    string            //是否开启来电显示
	Grpc       client.GrpcConfig //
}

type AuthContext struct {
	Service SuperAuthTokenService
}

//接口超类
type SuperAuthTokenService interface {
	//根据subject 信息 获取服务器颁发的 token
	AwardedToken(in *pb.AwardResponse) (*pb.AwardReplyResponse, error)
	//根据平台token 来获取刷新token
	RefreshToken(in *pb.AuthResponse) (*pb.ReplyResponse, error)
	//传递 token 到服务器进行验证
	AuthToken(in *pb.AuthResponse) (*pb.ReplyResponse, error)
	//授权之后 返回鉴权model
	AuthTokenResultModel(in *pb.AuthResponse) (*pb.ResultModelResponse, error)
}

type AuthGrpcClient struct {
	Client pb.AuthTokenServiceClient
}

type AuthServiceClient struct {
	RestUrl string
}

var AuthClient *AuthContext = nil

func InitService(conf AuthTokenConfig, metadata map[string]string) (bool, error) {
	AuthClient = new(AuthContext)
	if conf.AccessType == "grpc" {
		err := client.InitRPC(conf.Grpc, metadata)
		if err != nil {
			return false, err
		}
		AuthClient.Service = NewGrpc()
		return true, nil
	} else {
		AuthClient.Service = NewRest(conf.RestUrl)
	}
	return false, nil
}

func Release() {
	client.CloseRPC()
}
