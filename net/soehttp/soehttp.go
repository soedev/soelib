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
	"github.com/mitchellh/mapstructure"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/gin-gonic/gin"
	"github.com/soedev/soelib/common/utils"
)

//SoeRemoteService 在线服务
type SoeRemoteService struct {
	URL                string
	Token              string
	TenantID, ShopCode string //门店信息
	Context            *gin.Context
}

//SoeRestAPIException 异常
type SoeRestAPIException struct {
	Error     string `json:"error"`
	Exception string `json:"exception"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

//告警配置
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

//初始化http 配置信息
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

func (soeRemoteService *SoeRemoteService) Post(postBody *[]byte) ([]byte, error) {
	req, err := http.NewRequest("POST", soeRemoteService.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}
	return soeRemoteService.do(req, "RemotePost")
}

//get  get 请求
func (soeRemoteService *SoeRemoteService) Get(newReader io.Reader) ([]byte, error) {
	req, err := http.NewRequest("GET", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	return soeRemoteService.do(req, "RemoteGET")
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

func (soeRemoteService *SoeRemoteService) do(req *http.Request, operationName string) ([]byte, error) {
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
	if span, isOk := soeRemoteService.checkTracer(req, operationName); isOk {
		defer span.Finish()
		body, err := soeRemoteService.hyDo(client, req, operationName)
		if err != nil {
			span.SetTag("error", true)
			if err.Error() == "fallback" {
				span.LogKV("error", "发生熔断错误")
			} else {
				span.LogKV("error", err.Error())
			}
			return nil, err
		}
		return body, err
	} else {
		return soeRemoteService.hyDo(client, req, operationName)
	}
}

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

func (soeRemoteService *SoeRemoteService) hyDo(client *http.Client, req *http.Request, operationName string) (result []byte, err error) {
	hystrix.Do(operationName, func() error {
		var respond *http.Response
		respond, err = client.Do(req)
		if err != nil {
			return err
		}
		result, err = ioutil.ReadAll(respond.Body)
		if err != nil {
			return err
		}
		defer respond.Body.Close()
		return err
	}, func(err error) error {
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

//错误解析
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
		if err != nil {
			err = errors.New("服务器太忙了，请稍后再试！")
		}
		err = errors.New(soeRestAPIException.Message)
		return err
	}
	err = errors.New("服务器太忙了，请稍后再试！")
	return err
}
