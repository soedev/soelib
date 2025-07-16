package soehttp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type MockResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func TestSoeRemoteService_PostEntity(t *testing.T) {
	// 创建一个测试服务，返回固定 JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req MockRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: map[string]interface{}{
				"echo": req,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建 remote service 实例
	remote := Remote(server.URL)

	// 构造请求体
	input := MockRequest{Name: "Luchuang", Age: 18}
	var response SoeGoResponseVO

	err := remote.PostEntity(input, &response)
	if err != nil {
		t.Fatalf("PostEntity failed: %v", err)
	}

	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("response.Data is not a map, got: %T", response.Data)
	}

	echo, ok := dataMap["echo"].(map[string]interface{})
	if !ok {
		t.Fatalf("data.echo is not a map, got: %T", dataMap["echo"])
	}

	if echo["name"] != "Luchuang" || int(echo["age"].(float64)) != 18 {
		t.Errorf("Unexpected response data: %v", echo)
	}
}

func TestSoeRemoteService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: "pong",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	remote := Remote(server.URL)
	respBody, err := remote.Get(nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var result SoeGoResponseVO
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Data != "pong" {
		t.Errorf("Expected 'pong', got %v", result.Data)
	}
}
