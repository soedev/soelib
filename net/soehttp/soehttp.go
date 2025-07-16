package soehttp

/**
  soehttp  http 访问辅助类
*/

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/soedev/soelib/common/soelog"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/soedev/soelib/common/utils"
)

// SoeRemoteService 在线服务
type SoeRemoteService struct {
	URL                string
	Token              string
	TenantID, ShopCode string //门店信息
	Context            context.Context
	TimeOutSecond      string //超时时间,默认15秒
	Client             *http.Client
}

// SoeRestAPIException 异常
type SoeRestAPIException struct {
	Error     string `json:"error"`
	Exception string `json:"exception"`
	Message   string `json:"message"`
	Path      string `json:"path"`
	Data      string `json:"data"`
}

// SoeGoResponseVO go返回数据
type SoeGoResponseVO struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// 告警配置
type AlarmConfig struct {
	SendErrorToWx bool   //发送微信告警
	ChatID        string //微信群id
	ApiPath       string //微信发送路径
}

type SoeHTTPConfig struct {
	Alarm   AlarmConfig           //告警配置
	Hystrix hystrix.CommandConfig //熔断配置
}

var alarm = AlarmConfig{
	SendErrorToWx: false,
}

const (
	remotePost = "RemotePost"
	remoteGET  = "RemoteGET"
	remoteDel  = "RemoteDELETE"
)

// InitConfig 初始化http 配置信息
func InitConfig(config SoeHTTPConfig) {
	alarm = config.Alarm
	hystrix.ConfigureCommand(remotePost, config.Hystrix)
	hystrix.ConfigureCommand(remoteGET, config.Hystrix)
	hystrix.ConfigureCommand(remoteDel, config.Hystrix)
}
func Remote(url string, args ...string) *SoeRemoteService {
	soeRemoteService := SoeRemoteService{URL: url, Context: nil}
	switch len(args) {
	case 1:
		soeRemoteService.Token = args[0]
	case 2:
		soeRemoteService.Token = args[0]
		soeRemoteService.TenantID = args[1]
	case 3:
		soeRemoteService.Token = args[0]
		soeRemoteService.TenantID = args[1]
		soeRemoteService.ShopCode = args[2]
	case 4:
		soeRemoteService.Token = args[0]
		soeRemoteService.TenantID = args[1]
		soeRemoteService.ShopCode = args[2]
		soeRemoteService.TimeOutSecond = args[3]
		if soeRemoteService.TimeOutSecond == "" {
			soeRemoteService.TimeOutSecond = "15"
		}
	}
	soeRemoteService.Client = createDefaultClient(soeRemoteService.TimeOutSecond)
	return &soeRemoteService
}

func RemoteWithContent(content context.Context, url string, args ...string) *SoeRemoteService {
	soeRemoteService := SoeRemoteService{URL: url, Context: content}
	switch len(args) {
	case 1:
		soeRemoteService.Token = args[0]
	case 2:
		soeRemoteService.Token = args[0]
		soeRemoteService.TenantID = args[1]
	case 3:
		soeRemoteService.Token = args[0]
		soeRemoteService.TenantID = args[1]
		soeRemoteService.ShopCode = args[2]
	}
	soeRemoteService.Client = createDefaultClient(soeRemoteService.TimeOutSecond)
	return &soeRemoteService
}

func createDefaultClient(timeoutSeconds string) *http.Client {
	timeout := 15 // 默认15秒
	if timeoutSeconds != "" {
		if t, err := strconv.Atoi(timeoutSeconds); err == nil && t > 0 {
			timeout = t
		}
	}

	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: 1000,
	}

	return &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: tr,
	}
}

func (s *SoeRemoteService) WithClient(client *http.Client) *SoeRemoteService {
	s.Client = client
	return s
}

func (s *SoeRemoteService) NewPost(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequest("POST", s.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return s.NewDo(req, "RemotePost")
}

func (s *SoeRemoteService) NewGet(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("GET", s.URL, newReader)
	if err != nil {
		return nil, err
	}
	return s.NewDo(req, "RemoteGET")
}

func (s *SoeRemoteService) Post(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequest("POST", s.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return s.NewDo(req, "RemotePost")
}

func (s *SoeRemoteService) Get(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("GET", s.URL, newReader)
	if err != nil {
		return nil, err
	}
	return s.NewDo(req, "RemoteGET")
}
func (s *SoeRemoteService) Delete(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("DELETE", s.URL, newReader)
	if err != nil {
		return nil, err
	}
	return s.do(req, "RemoteDELETE")
}

func (s *SoeRemoteService) DeleteEntity(v interface{}) error {
	body, err := s.Delete(nil)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("为查询到任何信息")
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

func (s *SoeRemoteService) GetEntity(v interface{}) error {
	body, err := s.Get(nil)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("为查询到任何信息")
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

func (s *SoeRemoteService) PostEntity(v interface{}, r interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body, err := s.Post(&b)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("为查询到任何信息")
	}
	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}
	return nil
}

func (s *SoeRemoteService) do(req *http.Request, operationName string) (result []byte, err error) {
	client := s.Client
	s.setHeaders(req)

	err = hystrix.Do(operationName, func() error {
		var respond *http.Response
		respond, err = client.Do(req)
		if err != nil {
			return err
		}

		if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
			err = s.handleError(respond)
			return err
		}

		result, err = io.ReadAll(respond.Body)
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(respond.Body)
		return err
	}, func(err error) error {
		soelog.Logger.Info(req.URL.Host + ":" + req.URL.Path + "》》》》》》熔断降级：" + err.Error())
		if alarm.SendErrorToWx {
			if alarm.ChatID != "" && alarm.ApiPath != "" {
				content := fmt.Sprintf("GET 请求发生熔断错误！ URL:%s TenantID:%s", s.URL, s.TenantID)
				go func() {
					err := utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
					if err != nil {

					}
				}()
			}
		}
		err = errors.New("fallback")
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, err
}

// NewDo 去除熔断检测
func (s *SoeRemoteService) NewDo(req *http.Request, operationName string) (result []byte, err error) {
	client := s.Client
	s.setHeaders(req)
	respond, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("close http error:" + err.Error())
		}
	}(respond.Body)

	if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
		err = s.handleError(respond)
		return result, err
	}
	result, err = io.ReadAll(respond.Body)
	if err != nil {
		return result, err
	}
	return result, nil
}

// 错误解析
func (s *SoeRemoteService) handleError(resp *http.Response) (err error) {
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		err = errors.New(http.StatusText(resp.StatusCode))
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		err = errors.New(http.StatusText(resp.StatusCode))
		return err
	}
	switch t := data.(type) {
	case string:
		err = errors.New(t)
		return err
	case map[string]interface{}:
		soeRestAPIException := SoeRestAPIException{}
		err = mapstructure.Decode(t, &soeRestAPIException)
		if soeRestAPIException.Data != "" {
			return errors.New(fmt.Sprintf("%v", soeRestAPIException.Data))
		}
		if err != nil {
			err = errors.New("服务器太忙了，请稍后再试！")
		}
		err = errors.New(soeRestAPIException.Message)
		if soeRestAPIException.Message == "" {
			soeGoResponseVO := SoeGoResponseVO{}
			err = mapstructure.Decode(t, &soeGoResponseVO)
			if err != nil {
				err = errors.New("服务器太忙了，请稍后再试！")
			}
			if soeGoResponseVO.Code == 500 && reflect.TypeOf(soeGoResponseVO.Data).Kind() == reflect.String {
				err = errors.New(soeGoResponseVO.Data.(string))
			}
			if err == nil {
				err = errors.New("服务器太忙了，请稍后再试！")
			}
		}
		return err
	}
	err = errors.New("服务器太忙了，请稍后再试！")
	return err
}

func (s *SoeRemoteService) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if s.Token != "" {
		req.Header.Set("Authorization", s.Token)
	}
	if s.TenantID != "" {
		req.Header.Set("tenantId", s.TenantID)
	}
	if s.ShopCode != "" {
		req.Header.Set("shopCode", s.ShopCode)
	}
}
