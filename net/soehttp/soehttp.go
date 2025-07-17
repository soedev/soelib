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

type SoeRemoteService interface {
	Post(postBody *[]byte) ([]byte, error)
	Get(newReader io.Reader) ([]byte, error)
	PostEntity(v interface{}, r interface{}) error
	GetEntity(v interface{}) error
	Delete(newReader io.Reader) ([]byte, error)
	DeleteEntity(v interface{}) error
	WithClient(client *http.Client) SoeRemoteService
}

// RemoteOption 请求参数配置
type RemoteOption struct {
	URL           string
	Token         string
	TenantID      string
	ShopCode      string
	TimeoutSecond int
	Context       context.Context
}

// remoteServiceImpl 服务请求接口实现
type remoteServiceImpl struct {
	url      string
	token    string
	tenantID string
	shopCode string
	context  context.Context
	client   *http.Client
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

type AlarmConfig struct {
	SendErrorToWx bool   //发送微信告警
	ChatID        string //微信群id
	ApiPath       string //微信发送路径
}

type SoeHTTPConfig struct {
	Alarm         AlarmConfig           //告警配置
	Hystrix       hystrix.CommandConfig //熔断配置
	EnableHystrix bool
}

var (
	alarm         = AlarmConfig{SendErrorToWx: false}
	enableHystrix = false
)

const (
	remotePost = "RemotePost"
	remoteGET  = "RemoteGET"
	remoteDel  = "RemoteDELETE"
)

// InitConfig 初始化http 配置信息
func InitConfig(config SoeHTTPConfig) {
	alarm = config.Alarm
	enableHystrix = config.EnableHystrix
	hystrix.ConfigureCommand(remotePost, config.Hystrix)
	hystrix.ConfigureCommand(remoteGET, config.Hystrix)
	hystrix.ConfigureCommand(remoteDel, config.Hystrix)
}

var _ SoeRemoteService = (*remoteServiceImpl)(nil)

func NewRemote(opt RemoteOption) SoeRemoteService {
	if opt.Context == nil {
		opt.Context = context.Background()
	}
	if opt.TimeoutSecond <= 0 {
		opt.TimeoutSecond = 15
	}
	return &remoteServiceImpl{
		url:      opt.URL,
		token:    opt.Token,
		tenantID: opt.TenantID,
		shopCode: opt.ShopCode,
		context:  opt.Context,
		client:   createDefaultClient(opt.TimeoutSecond),
	}
}

// Remote 兼容系统老的写法
func Remote(url string, args ...string) SoeRemoteService {
	return RemoteWithContent(context.Background(), url, args...)
}

// RemoteWithContent 兼容系统老的写法
func RemoteWithContent(ctx context.Context, url string, args ...string) SoeRemoteService {
	timeout := 15 // 默认15秒
	soeRemoteService := &remoteServiceImpl{url: url, context: ctx}
	switch len(args) {
	case 1:
		soeRemoteService.token = args[0]
	case 2:
		soeRemoteService.token = args[0]
		soeRemoteService.tenantID = args[1]
	case 3:
		soeRemoteService.token = args[0]
		soeRemoteService.tenantID = args[1]
		soeRemoteService.shopCode = args[2]
	case 4:
		soeRemoteService.token = args[0]
		soeRemoteService.tenantID = args[1]
		soeRemoteService.shopCode = args[2]
		if args[3] != "" {
			if t, err := strconv.Atoi(args[3]); err == nil && t > 0 {
				timeout = t
			}
		}
	}
	soeRemoteService.client = createDefaultClient(timeout)
	return soeRemoteService
}

func createDefaultClient(timeoutSeconds int) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: 1000,
	}
	return &http.Client{
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
		Transport: tr,
	}
}

func (s *remoteServiceImpl) WithClient(client *http.Client) SoeRemoteService {
	s.client = client
	return s
}

func (s *remoteServiceImpl) Post(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(s.context, "POST", s.url, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return s.do(req, "RemotePost")
}

func (s *remoteServiceImpl) Get(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(s.context, "GET", s.url, newReader)
	if err != nil {
		return nil, err
	}
	return s.do(req, "RemoteGET")
}
func (s *remoteServiceImpl) Delete(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(s.context, "DELETE", s.url, newReader)
	if err != nil {
		return nil, err
	}
	return s.do(req, "RemoteDELETE")
}

func (s *remoteServiceImpl) DeleteEntity(v interface{}) error {
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

func (s *remoteServiceImpl) GetEntity(v interface{}) error {
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

func (s *remoteServiceImpl) PostEntity(v interface{}, r interface{}) error {
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

// NewDo 去除熔断检测
func (s *remoteServiceImpl) do(req *http.Request, operationName string) (result []byte, err error) {
	client := s.client
	s.setHeaders(req)
	if enableHystrix {
		// 启用熔断
		err = hystrix.Do(operationName, func() error {
			result, err = s.doRequest(client, req)
			return err
		}, func(e error) error {
			// 发生熔断处理逻辑
			s.sendFallbackAlert(req, e)
			return errors.New("fallback")
		})
	} else {
		result, err = s.doRequest(client, req)
	}
	return result, err
}

func (s *remoteServiceImpl) doRequest(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Printf("http do request 关闭流错误:%s \n", err.Error())
		}
	}(resp.Body)
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 207) {
		return nil, s.handleError(resp)
	}
	return io.ReadAll(resp.Body)
}

func (s *remoteServiceImpl) sendFallbackAlert(req *http.Request, err error) {
	soelog.Logger.Info(req.URL.Host + ":" + req.URL.Path + " 熔断降级：" + err.Error())
	if alarm.SendErrorToWx && alarm.ChatID != "" && alarm.ApiPath != "" {
		content := fmt.Sprintf("请求熔断错误！URL:%s TenantID:%s", s.url, s.tenantID)
		go func() {
			err = utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
			if err != nil {
				fmt.Printf("请求发生熔断错误，发送预警消息到企业微信失败:%s \n", err.Error())
			}
		}()
	}
}

// 错误解析
func (s *remoteServiceImpl) handleError(resp *http.Response) (err error) {
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

func (s *remoteServiceImpl) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if s.token != "" {
		req.Header.Set("Authorization", s.token)
	}
	if s.tenantID != "" {
		req.Header.Set("tenantId", s.tenantID)
	}
	if s.shopCode != "" {
		req.Header.Set("shopCode", s.shopCode)
	}
}
