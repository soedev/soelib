package auth2

import (
	"context"
	"fmt"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/net/grpc/client"
	pb "github.com/soedev/soelib/net/grpc/proto"
)

func NewGrpc() *AuthGrpcClient {
	instance := new(AuthGrpcClient)
	instance.Client = pb.NewAuthTokenServiceClient(client.RPCConn)
	return instance
}

func (s *AuthGrpcClient) AwardedToken(in *pb.AwardResponse) (*pb.AwardReplyResponse, error) {

	result, err := s.Client.AwardedToken(context.Background(), in)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("调用颁发平台token服务发生异常：%s", err.Error()))
	}
	return result, err
}

func (s *AuthGrpcClient) RefreshToken(in *pb.AuthResponse) (*pb.ReplyResponse, error) {
	result, err := s.Client.RefreshToken(context.Background(), in)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("刷新accessToken发生异常：%s", err.Error()))
	}
	return result, err
}

func (s *AuthGrpcClient) AuthToken(in *pb.AuthResponse) (*pb.ReplyResponse, error) {
	result, err := s.Client.AuthToken(context.Background(), in)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("调用鉴权服务发生异常：%s", err.Error()))
	}
	return result, err
}

func (s *AuthGrpcClient) AuthTokenResultModel(in *pb.AuthResponse) (*pb.ResultModelResponse, error) {
	result, err := s.Client.AuthTokenResultModel(context.Background(), in)
	if err != nil {
		soelog.Logger.Error(fmt.Sprintf("调用鉴权服务发生异常：%s", err.Error()))
	}
	return result, err
}
