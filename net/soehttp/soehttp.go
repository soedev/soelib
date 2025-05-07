package soehttp

/**
  soehttp  http 访问辅助类
*/

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/soedev/soelib/common/soelog"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/gin-gonic/gin"
	"github.com/soedev/soelib/common/utils"
)

// SoeRemoteService 在线服务
type SoeRemoteService struct {
	URL                string
	Token              string
	TenantID, ShopCode string //门店信息
	Context            *gin.Context
	TimeOutSecond      string //超时时间,默认15秒
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

// 初始化http 配置信息
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
	return &soeRemoteService
}

func RemoteWithContent(content *gin.Context, url string, args ...string) *SoeRemoteService {
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
	return &soeRemoteService
}

func (soeRemoteService *SoeRemoteService) NewPost(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequest("POST", soeRemoteService.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return soeRemoteService.NewDo(req, "RemotePost")
}

// get  get 请求
func (soeRemoteService *SoeRemoteService) NewGet(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("GET", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	return soeRemoteService.NewDo(req, "RemoteGET")
}

// NewPost 不加入熔断检测
func (soeRemoteService *SoeRemoteService) Post(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequest("POST", soeRemoteService.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return soeRemoteService.NewDo(req, "RemotePost")
}

// NewGet 不加入熔断检测
func (soeRemoteService *SoeRemoteService) Get(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("GET", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	return soeRemoteService.NewDo(req, "RemoteGET")
}
func (soeRemoteService *SoeRemoteService) Delete(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("DELETE", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	return soeRemoteService.do(req, "RemoteDELETE")
}

func (soeRemoteService *SoeRemoteService) DeleteEntity(v interface{}) error {
	body, err := soeRemoteService.Delete(nil)
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

func (soeRemoteService *SoeRemoteService) GetEntity(v interface{}) error {
	body, err := soeRemoteService.Get(nil)
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

func (soeRemoteService *SoeRemoteService) PostEntity(v interface{}, r interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body, err := soeRemoteService.Post(&b)
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

func (soeRemoteService *SoeRemoteService) do(req *http.Request, operationName string) (result []byte, err error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if soeRemoteService.Token != "" {
		req.Header.Set("Authorization", soeRemoteService.Token)
	}
	if soeRemoteService.TenantID != "" {
		req.Header.Set("tenantId", soeRemoteService.TenantID)
	}
	if soeRemoteService.ShopCode != "" {
		req.Header.Set("shopCode", soeRemoteService.ShopCode)
	}
	if span, isOk := soeRemoteService.checkTracer(req, soeRemoteService.URL); isOk {
		defer span.Finish()

		isTagError := false
		hystrix.Do(operationName, func() error {
			var respond *http.Response
			respond, err = client.Do(req)
			if err != nil {
				isTagError = true
				ext.Error.Set(span, true)
				span.LogKV("error", err.Error())
				return err
			}

			if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
				err = soeRemoteService.handleError(respond)
				isTagError = true
				ext.Error.Set(span, true)
				span.LogKV("error", err.Error())
				return err
			}

			result, err = ioutil.ReadAll(respond.Body)
			if err != nil {
				isTagError = true
				ext.Error.Set(span, true)
				span.LogKV("error", err.Error())
				return err
			}

			defer respond.Body.Close()
			return nil
		}, func(err error) error {
			if alarm.SendErrorToWx {
				if alarm.ChatID != "" && alarm.ApiPath != "" {
					content := fmt.Sprintf("GET 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
					go utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
				}
			}
			err = errors.New("fallback")
			if !isTagError {
				ext.Error.Set(span, true)
			}
			span.LogKV("error", "熔断错误")

			return err
		})
		return result, err
	} else {
		hystrix.Do(operationName, func() error {
			var respond *http.Response
			respond, err = client.Do(req)
			if err != nil {
				return err
			}

			if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
				err = soeRemoteService.handleError(respond)
				return err
			}

			result, err = ioutil.ReadAll(respond.Body)
			if err != nil {
				return err
			}
			defer respond.Body.Close()
			return err
		}, func(err error) error {
			soelog.Logger.Info(req.URL.Host + ":" + req.URL.Path + "》》》》》》熔断降级：" + err.Error())
			if alarm.SendErrorToWx {
				if alarm.ChatID != "" && alarm.ApiPath != "" {
					content := fmt.Sprintf("GET 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
					go utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
				}
			}
			err = errors.New("fallback")
			return err
		})
		return result, err
	}
}

// NewDo 去除熔断检测
func (soeRemoteService *SoeRemoteService) NewDo(req *http.Request, operationName string) (result []byte, err error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: 1000,
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}
	if soeRemoteService.TimeOutSecond != "" {
		timeOutSecond, _ := strconv.Atoi(soeRemoteService.TimeOutSecond)
		if timeOutSecond > 0 {
			client.Timeout = time.Duration(timeOutSecond) * time.Second
		}
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if soeRemoteService.Token != "" {
		req.Header.Set("Authorization", soeRemoteService.Token)
	}
	if soeRemoteService.TenantID != "" {
		req.Header.Set("tenantId", soeRemoteService.TenantID)
	}
	if soeRemoteService.ShopCode != "" {
		req.Header.Set("shopCode", soeRemoteService.ShopCode)
	}
	var respond *http.Response
	if span, isOk := soeRemoteService.checkTracer(req, soeRemoteService.URL); isOk {
		defer span.Finish()
		respond, err = client.Do(req)
		if respond != nil {
			defer respond.Body.Close()
		}
		if err != nil {
			ext.Error.Set(span, true)
			span.LogKV("error", err.Error())
			return result, err
		}

		if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
			err = soeRemoteService.handleError(respond)
			ext.Error.Set(span, true)
			span.LogKV("error", err.Error())
			return result, err
		}
		result, err = ioutil.ReadAll(respond.Body)
		if err != nil {
			ext.Error.Set(span, true)
			span.LogKV("error", err.Error())
			return result, err
		}
	} else {
		respond, err = client.Do(req)
		//重定向的错误时，respond将是 non-nil
		if respond != nil {
			defer respond.Body.Close()
		}
		if err != nil {
			return result, err
		}
		if !(respond.StatusCode >= 200 && respond.StatusCode <= 207) {
			err = soeRemoteService.handleError(respond)
			return result, err
		}

		result, err = ioutil.ReadAll(respond.Body)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

/*
//SendErrorToWx 发送错误日志到企业微信
func (soeRemoteService *SoeRemoteService) SendErrorToWx() {
	if alarm.SendErrorToWx {
		if alarm.ChatID != "" && alarm.ApiPath != "" {
			content := fmt.Sprintf(" 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
			go utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
		}
	}
}*/
//检测连接追踪
func (soeRemoteService *SoeRemoteService) checkTracer(req *http.Request, operationName string) (opentracing.Span, bool) {
	if soeRemoteService.Context != nil {
		tracer, isExists1 := soeRemoteService.Context.Get("Tracer")
		parentSpanContext, isExists2 := soeRemoteService.Context.Get("ParentSpanContext")
		if isExists1 && isExists2 {
			span := opentracing.StartSpan(
				operationName,
				opentracing.ChildOf(parentSpanContext.(opentracing.SpanContext)),
				opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
				ext.SpanKindRPCClient,
			)
			if soeRemoteService.TenantID != "" {
				span.SetTag("tenantId", soeRemoteService.TenantID)
			}
			if soeRemoteService.ShopCode != "" {
				span.SetTag("shopCode", soeRemoteService.ShopCode)
			}
			if soeRemoteService.URL != "" {
				span.SetTag("serviceUrl", soeRemoteService.URL)
			}
			injectErr := tracer.(opentracing.Tracer).Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
			if injectErr != nil {
				span.Finish()
				return nil, false
			}
			return span, true
		}
	}
	return nil, false
}

// func (soeRemoteService *SoeRemoteService) hyDo(client *http.Client, req *http.Request, operationName string) (result []byte, err error) {
// 	hystrix.Do(operationName, func() error {
// 		var respond *http.Response
// 		respond, err = client.Do(req)
// 		if err != nil {
// 			return err
// 		}
// 		result, err = ioutil.ReadAll(respond.Body)
// 		if err != nil {
// 			return err
// 		}
// 		defer respond.Body.Close()
// 		return err
// 	}, func(err error) error {
// 		if alarm.SendErrorToWx {
// 			if alarm.ChatID != "" && alarm.ApiPath != "" {
// 				content := fmt.Sprintf("GET 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
// 				go utils.SendMsgToWorkWx(alarm.ChatID, content, alarm.ApiPath, utils.WorkWxRestTokenStr)
// 			}
// 		}
// 		err = errors.New("fallback")
// 		return err
// 	})
// 	return result, err
// }

// 错误解析
func (soeRemoteService *SoeRemoteService) handleError(resp *http.Response) (err error) {
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		err = errors.New(http.StatusText(resp.StatusCode))
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
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
		err = errors.New(string(t))
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
