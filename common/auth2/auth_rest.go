package auth2

import (
	"errors"
	"fmt"
	pb "github.com/soedev/soelib/net/grpc/proto"
	"github.com/soedev/soelib/net/soehttp"
)

type AwardedTokenRes struct {
	Code int                   `json:"code"`
	Data pb.AwardReplyResponse `json:"data"`
	Msg  string                `json:"msg"`
}

type AuthResponse struct {
	Code int              `json:"code"`
	Data pb.ReplyResponse `json:"data"`
	Msg  string           `json:"msg"`
}
type AuthTokenResultModelRes struct {
	Code int                    `json:"code"`
	Data pb.ResultModelResponse `json:"data"`
	Msg  string                 `json:"msg"`
}

func NewRest(url string) *AuthServiceClient {
	instance := new(AuthServiceClient)
	instance.RestUrl = url
	return instance
}

func (s *AuthServiceClient) AwardedToken(in *pb.AwardResponse) (*pb.AwardReplyResponse, error) {
	url := s.RestUrl + "/api/token/award"
	var res AwardedTokenRes
	err := soehttp.Remote(url).PostEntity(in, &res)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用颁发平台token服务发生异常: %s ", err.Error()))
	}
	if res.Code != 200 {
		return nil, errors.New(fmt.Sprintf("调用颁发平台token服务发生异常:%s ", res.Msg))
	}
	return &res.Data, nil
}

func (s *AuthServiceClient) RefreshToken(in *pb.AuthResponse) (*pb.ReplyResponse, error) {
	url := s.RestUrl + "/api/token/refresh"
	var res AuthResponse
	err := soehttp.Remote(url).PostEntity(in, &res)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("刷新accessToken发生异常: %s ", err.Error()))
	}
	if res.Code != 200 {
		return nil, errors.New(fmt.Sprintf("刷新accessToken发生异常:%s ", res.Msg))
	}
	return &res.Data, nil
}

func (s *AuthServiceClient) AuthToken(in *pb.AuthResponse) (*pb.ReplyResponse, error) {
	url := s.RestUrl + "/api/token/auth"
	var res AuthResponse
	err := soehttp.Remote(url).PostEntity(in, &res)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用鉴权服务发生异常: %s ", err.Error()))
	}
	if res.Code != 200 {
		return nil, errors.New(fmt.Sprintf("调用鉴权服务发生异常:%s ", res.Msg))
	}
	return &res.Data, nil
}

func (s *AuthServiceClient) AuthTokenResultModel(in *pb.AuthResponse) (*pb.ResultModelResponse, error) {
	url := s.RestUrl + "/api/token/auth/result-model"
	var res AuthTokenResultModelRes
	err := soehttp.Remote(url).PostEntity(in, &res)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用鉴权服务发生异常: %s ", err.Error()))
	}
	if res.Code != 200 {
		return nil, errors.New(fmt.Sprintf("调用鉴权服务发生异常:%s ", res.Msg))
	}
	return &res.Data, nil
}
