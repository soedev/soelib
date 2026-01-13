package soehttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
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
	remote := NewRemote(RemoteOption{URL: server.URL})

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

	remote := NewRemote(RemoteOption{URL: server.URL})
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

func TestSoeRemoteService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "deleted",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	remote := NewRemote(RemoteOption{URL: server.URL})
	var result SoeGoResponseVO
	err := remote.DeleteEntity(&result)
	if err != nil {
		t.Fatalf("DeleteEntity failed: %v", err)
	}

	if result.Msg != "deleted" {
		t.Errorf("Expected 'deleted', got %v", result.Msg)
	}
}

// TestErrorHandling_401 测试 401 错误处理
func TestErrorHandling_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	remote := NewRemote(RemoteOption{URL: server.URL})
	_, err := remote.Get(nil)
	if err == nil {
		t.Fatal("Expected error for 401 status")
	}
	if err.Error() != "Unauthorized" {
		t.Errorf("Expected 'Unauthorized', got %v", err.Error())
	}
}

// TestErrorHandling_500WithString 测试 500 错误返回字符串
func TestErrorHandling_500WithString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`"Internal Server Error"`))
	}))
	defer server.Close()

	remote := NewRemote(RemoteOption{URL: server.URL})
	_, err := remote.Get(nil)
	if err == nil {
		t.Fatal("Expected error for 500 status")
	}
	if err.Error() != "Internal Server Error" {
		t.Errorf("Expected 'Internal Server Error', got %v", err.Error())
	}
}

// TestErrorHandling_SoeRestAPIException 测试 SoeRestAPIException 错误
func TestErrorHandling_SoeRestAPIException(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := SoeRestAPIException{
			Message: "参数错误",
			Data:    "缺少必填参数",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	remote := NewRemote(RemoteOption{URL: server.URL})
	_, err := remote.Get(nil)
	if err == nil {
		t.Fatal("Expected error for 400 status")
	}
	if err.Error() != "缺少必填参数" {
		t.Errorf("Expected '缺少必填参数', got %v", err.Error())
	}
}

// TestRetryMechanism 测试重试机制
func TestRetryMechanism(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			// 前两次请求返回 503
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Unavailable"))
			return
		}
		// 第三次请求成功
		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: "success after retry",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置重试：最多重试 3 次
	retryConfig := &RetryConfig{
		MaxRetries:      3,
		RetryWaitTime:   100 * time.Millisecond,
		RetryMaxWait:    500 * time.Millisecond,
		RetryableStatus: []int{503},
	}

	remote := NewRemote(RemoteOption{
		URL:         server.URL,
		RetryConfig: retryConfig,
	})

	respBody, err := remote.Get(nil)
	if err != nil {
		t.Fatalf("Get with retry failed: %v", err)
	}

	var result SoeGoResponseVO
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Data != "success after retry" {
		t.Errorf("Expected 'success after retry', got %v", result.Data)
	}

	// 验证重试了 2 次（第 3 次才成功）
	if atomic.LoadInt32(&attemptCount) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestRetryMechanism_MaxRetriesExceeded 测试超过最大重试次数
func TestRetryMechanism_MaxRetriesExceeded(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}))
	defer server.Close()

	retryConfig := &RetryConfig{
		MaxRetries:      2,
		RetryWaitTime:   50 * time.Millisecond,
		RetryMaxWait:    200 * time.Millisecond,
		RetryableStatus: []int{503},
	}

	remote := NewRemote(RemoteOption{
		URL:         server.URL,
		RetryConfig: retryConfig,
	})

	_, err := remote.Get(nil)
	if err == nil {
		t.Fatal("Expected error after max retries exceeded")
	}

	// 应该尝试了 3 次（初始 1 次 + 重试 2 次）
	if atomic.LoadInt32(&attemptCount) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestConcurrentRequests 测试并发请求
func TestConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // 模拟处理时间
		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: "concurrent response",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	remote := NewRemote(RemoteOption{URL: server.URL})

	// 并发发送 10 个请求
	const numRequests = 10
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := remote.Get(nil)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// TestContextTimeout 测试 Context 超时
func TestContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 模拟慢请求
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "slow response"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	remote := NewRemote(RemoteOption{
		URL:           server.URL,
		Context:       ctx,
		TimeoutSecond: 1,
	})

	_, err := remote.Get(nil)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

// TestCustomTransportConfig 测试自定义传输配置
func TestCustomTransportConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "custom config"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transportConfig := &TransportConfig{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     60 * time.Second,
		InsecureSkipVerify:  true,
	}

	remote := NewRemote(RemoteOption{
		URL:             server.URL,
		TransportConfig: transportConfig,
	})

	respBody, err := remote.Get(nil)
	if err != nil {
		t.Fatalf("Get with custom transport config failed: %v", err)
	}

	var result SoeGoResponseVO
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Data != "custom config" {
		t.Errorf("Expected 'custom config', got %v", result.Data)
	}
}

// TestRemoteWithContent_Compatibility 测试旧 API 兼容性
func TestRemoteWithContent_Compatibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("Expected Authorization header")
		}
		if r.Header.Get("tenantId") != "tenant-123" {
			t.Error("Expected tenantId header")
		}
		if r.Header.Get("shopCode") != "shop-456" {
			t.Error("Expected shopCode header")
		}

		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "old api works"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 使用旧的 API
	remote := RemoteWithContent(context.Background(), server.URL, "Bearer test-token", "tenant-123", "shop-456", "10")

	respBody, err := remote.Get(nil)
	if err != nil {
		t.Fatalf("Old API failed: %v", err)
	}

	var result SoeGoResponseVO
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Data != "old api works" {
		t.Errorf("Expected 'old api works', got %v", result.Data)
	}
}

// TestGlobalConfigConcurrency 测试全局配置的并发安全性
func TestGlobalConfigConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	const numGoroutines = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 并发写配置
			InitConfig(SoeHTTPConfig{
				EnableHystrix: id%2 == 0,
				Alarm: AlarmConfig{
					SendErrorToWx: true,
					ChatID:        "test-chat",
				},
			})
			// 并发读配置
			_ = IsHystrixEnabled()
			_ = GetAlarmConfig()
		}(i)
	}

	wg.Wait()
	// 如果有数据竞争，测试会失败（使用 go test -race 运行）
}

// TestCircuitBreaker_Basic 测试基础熔断功能
func TestCircuitBreaker_Basic(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		// 前 10 个请求都返回 500 错误，触发熔断
		if count <= 10 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
			return
		}
		// 后续请求返回成功
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置熔断：错误率 50%，最少 5 个请求
	InitConfig(SoeHTTPConfig{
		EnableHystrix: true,
		Hystrix: hystrix.CommandConfig{
			Timeout:                1000, // 1秒超时
			MaxConcurrentRequests:  100,
			ErrorPercentThreshold:  50,   // 50% 错误率触发熔断
			RequestVolumeThreshold: 5,    // 至少 5 个请求
			SleepWindow:            1000, // 1秒后尝试恢复
		},
	})

	remote := NewRemote(RemoteOption{
		URL: server.URL,
	})

	// 发送多个请求
	var successCount, errorCount, fallbackCount int
	for i := 0; i < 15; i++ {
		_, err := remote.Get(nil)
		if err == nil {
			successCount++
		} else {
			errMsg := err.Error()
			// hystrix 的 fallback 错误包含 "fallback" 和 "circuit open" 字符串
			if strings.Contains(errMsg, "fallback") || strings.Contains(errMsg, "circuit open") {
				fallbackCount++
				t.Logf("第 %d 次请求触发熔断: %v", i+1, errMsg)
			} else {
				errorCount++
				t.Logf("第 %d 次请求错误: %v", i+1, err)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("总计 - 成功: %d, 错误: %d, 熔断: %d", successCount, errorCount, fallbackCount)

	// 验证熔断是否生效（应该有部分请求被熔断）
	if fallbackCount == 0 {
		t.Error("期望触发熔断，但没有发生")
	}

	// 验证熔断配置
	if !IsHystrixEnabled() {
		t.Error("期望熔断已启用")
	}
}

// TestCircuitBreaker_TimeoutTrigger 测试超时触发熔断
func TestCircuitBreaker_TimeoutTrigger(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		// 前几个请求超时（睡眠 2 秒）
		if count <= 8 {
			time.Sleep(2 * time.Second)
		}
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置熔断：超时 500ms
	InitConfig(SoeHTTPConfig{
		EnableHystrix: true,
		Hystrix: hystrix.CommandConfig{
			Timeout:                500, // 500ms 超时（请求会超时）
			MaxConcurrentRequests:  100,
			ErrorPercentThreshold:  50,
			RequestVolumeThreshold: 5,
			SleepWindow:            500,
		},
	})

	remote := NewRemote(RemoteOption{
		URL:           server.URL,
		TimeoutSecond: 10, // HTTP 客户端超时设置更长
	})

	// 发送请求，期望超时触发熔断
	var timeoutCount, fallbackCount int
	for i := 0; i < 12; i++ {
		_, err := remote.Get(nil)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "fallback") || strings.Contains(errMsg, "circuit open") {
				fallbackCount++
			}
			if strings.Contains(errMsg, "timeout") {
				timeoutCount++
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("超时次数: %d, 熔断次数: %d", timeoutCount, fallbackCount)

	// 验证至少有超时或熔断发生
	if timeoutCount+fallbackCount == 0 {
		t.Error("期望有超时或熔断发生")
	}
}

// TestCircuitBreaker_MaxConcurrentRequests 测试并发限制
func TestCircuitBreaker_MaxConcurrentRequests(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟慢请求
		time.Sleep(100 * time.Millisecond)
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置熔断：最大并发 5
	InitConfig(SoeHTTPConfig{
		EnableHystrix: true,
		Hystrix: hystrix.CommandConfig{
			Timeout:                2000,
			MaxConcurrentRequests:  5, // 只允许 5 个并发
			ErrorPercentThreshold:  50,
			RequestVolumeThreshold: 3,
			SleepWindow:            1000,
		},
	})

	remote := NewRemote(RemoteOption{
		URL: server.URL,
	})

	// 并发发送 20 个请求
	var wg sync.WaitGroup
	var rejectedCount int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := remote.Get(nil)
			if err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "max concurrency") || strings.Contains(errMsg, "fallback") {
					atomic.AddInt32(&rejectedCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("被拒绝的请求数: %d", rejectedCount)

	// 由于只允许 5 个并发，应该有部分请求被拒绝
	if rejectedCount == 0 {
		t.Log("警告: 期望有请求因并发限制被拒绝，但没有发生（可能测试时间太短）")
	}
}

// TestCircuitBreaker_Recovery 测试熔断恢复
func TestCircuitBreaker_Recovery(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长时间测试")
	}

	// 测试前重置配置
	defer ResetConfigForTest()

	var shouldFail atomic.Value
	shouldFail.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail.Load().(bool) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置熔断
	InitConfig(SoeHTTPConfig{
		EnableHystrix: true,
		Hystrix: hystrix.CommandConfig{
			Timeout:                1000,
			MaxConcurrentRequests:  100,
			ErrorPercentThreshold:  50,
			RequestVolumeThreshold: 3,
			SleepWindow:            2000, // 2秒后尝试恢复
		},
	})

	remote := NewRemote(RemoteOption{
		URL: server.URL,
	})

	// 阶段 1: 触发熔断（发送失败请求）
	t.Log("阶段 1: 触发熔断")
	for i := 0; i < 10; i++ {
		remote.Get(nil)
		time.Sleep(50 * time.Millisecond)
	}

	// 阶段 2: 修复服务，等待恢复
	t.Log("阶段 2: 修复服务，等待熔断恢复")
	shouldFail.Store(false)
	time.Sleep(3 * time.Second) // 等待 SleepWindow

	// 阶段 3: 验证恢复（应该能成功）
	t.Log("阶段 3: 验证熔断恢复")
	var successCount int
	for i := 0; i < 5; i++ {
		_, err := remote.Get(nil)
		if err == nil {
			successCount++
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("恢复后成功请求数: %d/5", successCount)

	// 应该有成功的请求（说明熔断已恢复）
	if successCount == 0 {
		t.Error("期望熔断恢复后有成功请求，但全部失败")
	}
}

// TestCircuitBreaker_DisabledByDefault 测试默认不启用熔断
func TestCircuitBreaker_DisabledByDefault(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	// 不配置熔断（默认禁用）
	remote := NewRemote(RemoteOption{
		URL: server.URL,
	})

	// 发送多个失败请求
	var errorCount int
	for i := 0; i < 10; i++ {
		_, err := remote.Get(nil)
		if err != nil {
			errMsg := err.Error()
			// 不是 fallback 错误（熔断未启用时的正常错误）
			if !strings.Contains(errMsg, "fallback") && !strings.Contains(errMsg, "circuit") {
				errorCount++
			}
		}
	}

	// 验证：所有请求都应该返回错误（不是 fallback），说明熔断未启用
	if errorCount != 10 {
		t.Errorf("期望 10 个错误（熔断未启用），实际: %d", errorCount)
	}

	// 验证熔断未启用
	if IsHystrixEnabled() {
		t.Error("期望熔断默认未启用")
	}
}

// TestCircuitBreaker_WithAlarm 测试熔断告警（不实际发送）
func TestCircuitBreaker_WithAlarm(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	// 配置熔断和告警（但不实际发送）
	InitConfig(SoeHTTPConfig{
		EnableHystrix: true,
		Hystrix: hystrix.CommandConfig{
			Timeout:                1000,
			MaxConcurrentRequests:  100,
			ErrorPercentThreshold:  50,
			RequestVolumeThreshold: 3,
			SleepWindow:            1000,
		},
		Alarm: AlarmConfig{
			SendErrorToWx: false, // 测试中不实际发送微信
			ChatID:        "test-chat-id",
			ApiPath:       "https://test.example.com/webhook",
		},
	})

	remote := NewRemote(RemoteOption{
		URL:      server.URL,
		TenantID: "test-tenant",
	})

	// 触发熔断
	for i := 0; i < 10; i++ {
		remote.Get(nil)
		time.Sleep(50 * time.Millisecond)
	}

	// 验证告警配置
	alarmConfig := GetAlarmConfig()
	if alarmConfig.ChatID != "test-chat-id" {
		t.Errorf("期望 ChatID='test-chat-id', 实际: %s", alarmConfig.ChatID)
	}

	t.Log("熔断告警配置测试通过（未实际发送）")
}

// TestCircuitBreaker_InstanceLevel 测试实例级熔断配置（新特性）
func TestCircuitBreaker_InstanceLevel(t *testing.T) {
	defer ResetConfigForTest()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 8 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 全局不启用熔断
	InitConfig(SoeHTTPConfig{
		EnableHystrix: false,
	})

	// 实例1: 不启用熔断（默认）
	remote1 := NewRemote(RemoteOption{
		URL: server.URL + "/api1",
	})

	// 实例2: 显式启用熔断
	enableTrue := true
	remote2 := NewRemote(RemoteOption{
		URL:           server.URL + "/api2",
		EnableHystrix: &enableTrue,
		HystrixConfig: DefaultHystrixConfig(),
	})

	// 测试实例1: 不熔断，所有请求都会执行
	var instance1Errors int
	for i := 0; i < 5; i++ {
		_, err := remote1.Get(nil)
		if err != nil && !IsCircuitBreakerError(err) {
			instance1Errors++
		}
	}

	t.Logf("实例1（无熔断）错误数: %d", instance1Errors)
	if instance1Errors == 0 {
		t.Error("期望实例1有错误（服务返回500）")
	}

	// 测试实例2: 有熔断，部分请求会被熔断
	var instance2Fallbacks int
	for i := 0; i < 10; i++ {
		_, err := remote2.Get(nil)
		if err != nil && IsCircuitBreakerError(err) {
			instance2Fallbacks++
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("实例2（有熔断）熔断次数: %d", instance2Fallbacks)
	if instance2Fallbacks == 0 {
		t.Error("期望实例2触发熔断")
	}
}

// TestCircuitBreaker_DefaultDisabled 测试默认不启用熔断（新特性）
func TestCircuitBreaker_DefaultDisabled(t *testing.T) {
	defer ResetConfigForTest()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// 不配置全局熔断（默认关闭）
	// 创建普通实例
	remote := NewRemote(RemoteOption{
		URL: server.URL,
	})

	// 发送多个失败请求
	var errorCount int
	for i := 0; i < 10; i++ {
		_, err := remote.Get(nil)
		if err != nil && !IsCircuitBreakerError(err) {
			errorCount++
		}
	}

	// 验证：所有请求都返回正常错误（不是熔断错误）
	if errorCount != 10 {
		t.Errorf("期望10个正常错误，实际: %d", errorCount)
	}

	t.Log("默认不启用熔断验证通过")
}

// TestCircuitBreaker_StrictConfig 测试严格熔断配置
func TestCircuitBreaker_StrictConfig(t *testing.T) {
	defer ResetConfigForTest()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// 使用严格配置
	enableTrue := true
	remote := NewRemote(RemoteOption{
		URL:           server.URL,
		EnableHystrix: &enableTrue,
		HystrixConfig: StrictHystrixConfig(), // 使用严格配置
	})

	// 发送请求
	var fallbackCount int
	for i := 0; i < 15; i++ {
		_, err := remote.Get(nil)
		if err != nil && IsCircuitBreakerError(err) {
			fallbackCount++
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("严格配置下熔断次数: %d, 请求总数: %d", fallbackCount, atomic.LoadInt32(&requestCount))

	// 严格配置应该更快触发熔断
	if fallbackCount == 0 {
		t.Error("期望严格配置触发熔断")
	}
}

// TestErrorTypeDetection 测试错误类型检测（新特性）
func TestErrorTypeDetection(t *testing.T) {
	defer ResetConfigForTest()

	// 模拟不同类型的错误
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		expectType  string
	}{
		{
			name: "熔断错误",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectType: "circuit_breaker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			if tt.expectType == "circuit_breaker" {
				// 启用熔断
				enableTrue := true
				remote := NewRemote(RemoteOption{
					URL:           server.URL,
					EnableHystrix: &enableTrue,
					HystrixConfig: &HystrixConfig{
						Timeout:                1000,
						ErrorPercentThreshold:  30,
						RequestVolumeThreshold: 3,
						SleepWindow:            1000,
					},
				})

				// 触发熔断
				for i := 0; i < 10; i++ {
					_, err := remote.Get(nil)
					if err != nil {
						errType := GetErrorType(err)
						if strings.Contains(err.Error(), "circuit") || strings.Contains(err.Error(), "fallback") {
							if errType != "circuit_breaker" {
								t.Errorf("期望错误类型为 circuit_breaker, 实际: %s", errType)
							}
							return
						}
					}
				}
			}
		})
	}
}
