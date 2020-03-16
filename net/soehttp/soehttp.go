package soehttp

/**
  soehttp  http 访问辅助类
*/

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
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
}

//SoeRestAPIException 异常
type SoeRestAPIException struct {
	Error     string `json:"error"`
	Exception string `json:"exception"`
	Message   string `json:"message"`
	Path      string `json:"path"`
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
func (soeRemoteService *SoeRemoteService) Post(postBody *[]byte) ([]byte, error) {
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

//Get get
func (soeRemoteService *SoeRemoteService) Get(newReader io.Reader) ([]byte, error) {
	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
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
	soeRemoteService := SoeRemoteService{URL: url}
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
