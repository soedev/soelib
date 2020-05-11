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
	"github.com/afex/hystrix-go/hystrix"
	"github.com/soedev/soelib/common/utils"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
)

//SoeRemoteService 在线服务
type SoeRemoteService struct {
	URL                string
	Token              string
	TenantID, ShopCode string //门店信息
	UseHystrix         bool
}

//SoeRestAPIException 异常
type SoeRestAPIException struct {
	Error     string `json:"error"`
	Exception string `json:"exception"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

type hystrixFallMsgSendConfig struct {
	IsSendToWx bool
	ChatID     string
	ApiPath    string
}

//熔断配置方案
var config = hystrix.CommandConfig{
	Timeout:                5000, //执行command的超时时间(毫秒)
	MaxConcurrentRequests:  8,    //command的最大并发量
	SleepWindow:            1000, //过多长时间，熔断器再次检测是否开启。单位毫秒
	ErrorPercentThreshold:  30,   //错误率 请求数量大于等于RequestVolumeThreshold并且错误率到达这个百分比后就会启动
	RequestVolumeThreshold: 5,    //请求阈值(一个统计窗口10秒内请求数量)  熔断器是否打开首先要满足这个条件；这里的设置表示至少有5个请求才进行ErrorPercentThreshold错误百分比计算
}

var HttpErrorSendConfig = hystrixFallMsgSendConfig{
	IsSendToWx: false,
}

//默认熔断配置
func init() {
	hystrix.ConfigureCommand("get", config)
	hystrix.ConfigureCommand("post", config)
}

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

//Post post
func (soeRemoteService *SoeRemoteService) Post(postBody *[]byte) (result []byte, err error) {
	//if soeRemoteService.UseHystrix {
		hystrix.Do("post", func() error {
			result, err = soeRemoteService.post(postBody)
			return err
		}, func(err error) error {
			if HttpErrorSendConfig.IsSendToWx {
				if HttpErrorSendConfig.ChatID != "" && HttpErrorSendConfig.ApiPath != "" {
					content := fmt.Sprintf("POST 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
					go utils.SendMsgToWorkWx(HttpErrorSendConfig.ChatID, content, HttpErrorSendConfig.ApiPath, utils.WorkWxRestTokenStr)
				}
			}
			err = errors.New("fallback")
			return err
		})
		return result, err
	//} else {
	//	return soeRemoteService.post(postBody)
	//}
}

//Get get
func (soeRemoteService *SoeRemoteService) Get(newReader io.Reader) (result []byte, err error) {
	//if soeRemoteService.UseHystrix {
		hystrix.Do("get", func() error {
			result, err = soeRemoteService.get(newReader)
			return err
		}, func(err error) error {
			if HttpErrorSendConfig.IsSendToWx {
				if HttpErrorSendConfig.ChatID != "" && HttpErrorSendConfig.ApiPath != "" {
					content := fmt.Sprintf("GET 请求发生熔断错误！ URL:%s TenantID:%s", soeRemoteService.URL, soeRemoteService.TenantID)
					go utils.SendMsgToWorkWx(HttpErrorSendConfig.ChatID, content, HttpErrorSendConfig.ApiPath, utils.WorkWxRestTokenStr)
				}
			}
			err = errors.New("fallback")
			return err
		})
		return result, err
	//} else {
	//	return soeRemoteService.get(newReader)
	//}
}

func (soeRemoteService *SoeRemoteService) post(postBody *[]byte) ([]byte, error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}
	req, err := http.NewRequest("POST", soeRemoteService.URL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	if soeRemoteService.Token != "" {
		req.Header.Set("Authorization", soeRemoteService.Token)
	}
	if soeRemoteService.TenantID != "" {
		req.Header.Set("tenantId", soeRemoteService.TenantID)
	}
	if soeRemoteService.ShopCode != "" {
		req.Header.Set("shopCode", soeRemoteService.ShopCode)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return body, err
}

func (soeRemoteService *SoeRemoteService) get(newReader io.Reader) ([]byte, error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:  5 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}

	req, err := http.NewRequest("GET", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return body, err
}

func (soeRemoteService *SoeRemoteService) Delete(newReader io.Reader) ([]byte, error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}
	req, err := http.NewRequest("DELETE", soeRemoteService.URL, newReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
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
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 207) {
		err = soeRemoteService.handleError(resp)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return body, err
}

func Remote(url string, args ...string) *SoeRemoteService {
	soeRemoteService := SoeRemoteService{URL: url, UseHystrix: false}
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

func RemoteUseHystrix(url string, args ...string) *SoeRemoteService {
	soeRemoteService := SoeRemoteService{URL: url, UseHystrix: true}
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
