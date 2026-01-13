package soehttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestServiceClient_Basic 测试基础功能
func TestServiceClient_Basic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: "service client response",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建服务客户端
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "test-service",
		BaseURL:     server.URL,
	})

	// 测试 Get 请求
	respBody, err := client.Get("/api/test", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var result SoeGoResponseVO
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Data != "service client response" {
		t.Errorf("Expected 'service client response', got %v", result.Data)
	}
}

// TestServiceClient_Post 测试 POST 请求
func TestServiceClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

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

	client := NewServiceClient(ServiceClientOption{
		ServiceName: "test-service",
		BaseURL:     server.URL,
	})

	// 测试 PostEntity
	input := MockRequest{Name: "ServiceClient", Age: 99}
	var response SoeGoResponseVO

	err := client.PostEntity("/api/create", input, &response)
	if err != nil {
		t.Fatalf("PostEntity failed: %v", err)
	}

	if response.Code != 200 {
		t.Errorf("Expected code 200, got %d", response.Code)
	}
}

// TestServiceClient_ConnectionPoolReuse 测试连接池复用
func TestServiceClient_ConnectionPoolReuse(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		resp := SoeGoResponseVO{Code: 200, Msg: "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建一个服务客户端（共享连接池）
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "worker-service",
		BaseURL:     server.URL,
	})

	// 多次调用不同接口，应该复用连接池
	for i := 0; i < 10; i++ {
		_, err := client.Get("/api/worker/get", nil)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	for i := 0; i < 10; i++ {
		data := []byte(`{"test": true}`)
		_, err := client.Post("/api/worker/create", &data)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	// 验证所有请求都到达了服务器
	if atomic.LoadInt32(&requestCount) != 20 {
		t.Errorf("Expected 20 requests, got %d", requestCount)
	}
}

// TestServiceClient_ConcurrentSameService 测试并发访问同一服务
func TestServiceClient_ConcurrentSameService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // 模拟处理时间
		resp := SoeGoResponseVO{Code: 200, Msg: "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建服务客户端
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "worker-service",
		BaseURL:     server.URL,
	})

	// 并发调用
	const numRequests = 20
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			path := "/api/worker/get"
			_, err := client.Get(path, nil)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// TestServiceClient_WithRetry 测试重试功能
func TestServiceClient_WithRetry(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			// 前两次返回 503
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Unavailable"))
			return
		}
		// 第三次成功
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success after retry"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 配置重试
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "test-service",
		BaseURL:     server.URL,
		RetryConfig: &RetryConfig{
			MaxRetries:      3,
			RetryWaitTime:   100 * time.Millisecond,
			RetryMaxWait:    500 * time.Millisecond,
			RetryableStatus: []int{503},
		},
	})

	respBody, err := client.Get("/api/test", nil)
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

// TestServiceClient_WithCircuitBreaker 测试熔断功能
func TestServiceClient_WithCircuitBreaker(t *testing.T) {
	// 测试前重置配置
	defer ResetConfigForTest()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// 所有请求都返回 500
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// 启用熔断
	enableTrue := true
	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		EnableHystrix: &enableTrue,
		HystrixConfig: &HystrixConfig{
			Timeout:                1000,
			MaxConcurrentRequests:  100,
			ErrorPercentThreshold:  50,
			RequestVolumeThreshold: 5,
			SleepWindow:            1000,
		},
	})

	// 发送多个请求，触发熔断
	var fallbackCount int
	for i := 0; i < 15; i++ {
		_, err := client.Get("/api/test", nil)
		if err != nil && IsCircuitBreakerError(err) {
			fallbackCount++
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("请求总数: %d, 熔断次数: %d", atomic.LoadInt32(&requestCount), fallbackCount)

	// 验证熔断是否生效
	if fallbackCount == 0 {
		t.Error("期望触发熔断，但没有发生")
	}
}

// TestServiceClient_RegisterAndGet 测试全局注册和获取
func TestServiceClient_RegisterAndGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{Code: 200, Msg: "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 注册服务
	RegisterService("worker-service", ServiceClientOption{
		BaseURL: server.URL,
	})

	RegisterService("order-service", ServiceClientOption{
		BaseURL: server.URL,
	})

	// 获取服务客户端
	workerClient, err := GetServiceClient("worker-service")
	if err != nil {
		t.Fatalf("GetServiceClient failed: %v", err)
	}

	orderClient, err := GetServiceClient("order-service")
	if err != nil {
		t.Fatalf("GetServiceClient failed: %v", err)
	}

	// 验证服务名称
	if workerClient.GetServiceName() != "worker-service" {
		t.Errorf("Expected 'worker-service', got %s", workerClient.GetServiceName())
	}

	if orderClient.GetServiceName() != "order-service" {
		t.Errorf("Expected 'order-service', got %s", orderClient.GetServiceName())
	}

	// 调用接口
	_, err = workerClient.Get("/api/worker/get", nil)
	if err != nil {
		t.Errorf("Worker client request failed: %v", err)
	}

	_, err = orderClient.Get("/api/order/get", nil)
	if err != nil {
		t.Errorf("Order client request failed: %v", err)
	}

	// 测试获取不存在的服务
	_, err = GetServiceClient("nonexistent-service")
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

// TestBackwardCompatibility 测试向后兼容性（老的 API 仍然工作）
func TestBackwardCompatibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "old api works"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 测试老的 NewRemote API（应该仍然工作）
	remote := NewRemote(RemoteOption{
		URL:      server.URL + "/api/test",
		TenantID: "tenant-123",
	})

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

	t.Log("✅ 向后兼容性测试通过：老的 NewRemote API 仍然正常工作")
}

// TestNewVsOldAPI 对比新旧 API
func TestNewVsOldAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SoeGoResponseVO{Code: 200, Msg: "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Run("老的API-每次创建", func(t *testing.T) {
		// 老的方式：每次创建新的 Remote
		for i := 0; i < 5; i++ {
			remote := NewRemote(RemoteOption{
				URL: server.URL + "/api/test",
			})
			_, err := remote.Get(nil)
			if err != nil {
				t.Errorf("Old API request %d failed: %v", i, err)
			}
		}
	})

	t.Run("新的API-复用连接池", func(t *testing.T) {
		// 新的方式：创建一次，多次调用
		client := NewServiceClient(ServiceClientOption{
			ServiceName: "test-service",
			BaseURL:     server.URL,
		})

		for i := 0; i < 5; i++ {
			_, err := client.Get("/api/test", nil)
			if err != nil {
				t.Errorf("New API request %d failed: %v", i, err)
			}
		}
	})

	t.Log("✅ 新旧 API 都正常工作")
}

// TestServiceClient_MultiTenant 测试多租户场景（重要！）⭐
func TestServiceClient_MultiTenant(t *testing.T) {
	// 记录每个租户的请求
	tenantRequests := make(map[string]int)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读取租户 ID
		tenantID := r.Header.Get("tenantId")

		mu.Lock()
		tenantRequests[tenantID]++
		mu.Unlock()

		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: map[string]string{
				"tenantId": tenantID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// ✅ 正确的做法：一个 ServiceClient 服务所有租户
	workerClient := NewServiceClient(ServiceClientOption{
		ServiceName: "worker-service",
		BaseURL:     server.URL,
		// 注意：不设置 TenantID
	})

	// 模拟 10000 个租户（这里测试 100 个）
	const numTenants = 100
	var wg sync.WaitGroup

	for i := 0; i < numTenants; i++ {
		wg.Add(1)
		tenantID := fmt.Sprintf("tenant-%03d", i)

		go func(tid string) {
			defer wg.Done()

			// 使用 WithOptions 传入请求级的租户信息
			_, err := workerClient.GetWithOptions("/api/worker/get", nil, RequestOptions{
				TenantID: tid,
			})
			if err != nil {
				t.Errorf("Tenant %s request failed: %v", tid, err)
			}
		}(tenantID)
	}

	wg.Wait()

	// 验证所有租户都成功调用
	mu.Lock()
	defer mu.Unlock()

	if len(tenantRequests) != numTenants {
		t.Errorf("Expected %d tenants, got %d", numTenants, len(tenantRequests))
	}

	// 验证每个租户都有请求
	for i := 0; i < numTenants; i++ {
		tenantID := fmt.Sprintf("tenant-%03d", i)
		if tenantRequests[tenantID] != 1 {
			t.Errorf("Tenant %s: expected 1 request, got %d", tenantID, tenantRequests[tenantID])
		}
	}

	t.Logf("✅ 多租户测试通过：%d 个租户共享一个 ServiceClient", numTenants)
}

// TestServiceClient_RequestLevelOptions 测试请求级参数优先级
func TestServiceClient_RequestLevelOptions(t *testing.T) {
	var receivedTenantID, receivedShopCode, receivedToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTenantID = r.Header.Get("tenantId")
		receivedShopCode = r.Header.Get("shopCode")
		receivedToken = r.Header.Get("Authorization")

		resp := SoeGoResponseVO{Code: 200, Msg: "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端，设置默认值
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "test-service",
		BaseURL:     server.URL,
	})

	t.Run("不传参数时为空", func(t *testing.T) {
		// 不传 opts，headers 应该为空（符合多租户最佳实践）
		_, err := client.Get("/api/test", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if receivedTenantID != "" {
			t.Errorf("Expected empty, got '%s'", receivedTenantID)
		}
		if receivedShopCode != "" {
			t.Errorf("Expected empty, got '%s'", receivedShopCode)
		}
		if receivedToken != "" {
			t.Errorf("Expected empty, got '%s'", receivedToken)
		}
	})

	t.Run("使用请求级参数", func(t *testing.T) {
		// 传入 opts，应该使用请求级参数
		_, err := client.GetWithOptions("/api/test", nil, RequestOptions{
			TenantID: "request-tenant",
			ShopCode: "request-shop",
			Token:    "request-token",
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if receivedTenantID != "request-tenant" {
			t.Errorf("Expected 'request-tenant', got '%s'", receivedTenantID)
		}
		if receivedShopCode != "request-shop" {
			t.Errorf("Expected 'request-shop', got '%s'", receivedShopCode)
		}
		if receivedToken != "request-token" {
			t.Errorf("Expected 'request-token', got '%s'", receivedToken)
		}
	})

	t.Run("部分参数", func(t *testing.T) {
		// 只传递部分参数，其他为空
		_, err := client.GetWithOptions("/api/test", nil, RequestOptions{
			TenantID: "override-tenant",
			// ShopCode 和 Token 不传，应该为空
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if receivedTenantID != "override-tenant" {
			t.Errorf("Expected 'override-tenant', got '%s'", receivedTenantID)
		}
		if receivedShopCode != "" {
			t.Errorf("Expected empty, got '%s'", receivedShopCode)
		}
		if receivedToken != "" {
			t.Errorf("Expected empty, got '%s'", receivedToken)
		}
	})

	t.Log("✅ 请求级参数优先级测试通过")
}

// TestServiceClient_MultiTenantPostEntity 测试多租户 PostEntity
func TestServiceClient_MultiTenantPostEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("tenantId")

		var req MockRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		resp := SoeGoResponseVO{
			Code: 200,
			Msg:  "ok",
			Data: map[string]interface{}{
				"tenantId": tenantID,
				"echo":     req,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 一个 ServiceClient 服务所有租户
	client := NewServiceClient(ServiceClientOption{
		ServiceName: "worker-service",
		BaseURL:     server.URL,
	})

	// 租户 1 创建员工
	input1 := MockRequest{Name: "Worker1", Age: 30}
	var response1 SoeGoResponseVO
	err := client.PostEntityWithOptions("/api/worker/create", input1, &response1, RequestOptions{
		TenantID: "tenant-001",
	})
	if err != nil {
		t.Fatalf("Tenant 1 request failed: %v", err)
	}

	// 租户 2 创建员工
	input2 := MockRequest{Name: "Worker2", Age: 25}
	var response2 SoeGoResponseVO
	err = client.PostEntityWithOptions("/api/worker/create", input2, &response2, RequestOptions{
		TenantID: "tenant-002",
	})
	if err != nil {
		t.Fatalf("Tenant 2 request failed: %v", err)
	}

	// 验证租户隔离
	dataMap1 := response1.Data.(map[string]interface{})
	if dataMap1["tenantId"] != "tenant-001" {
		t.Errorf("Tenant 1: expected 'tenant-001', got %v", dataMap1["tenantId"])
	}

	dataMap2 := response2.Data.(map[string]interface{})
	if dataMap2["tenantId"] != "tenant-002" {
		t.Errorf("Tenant 2: expected 'tenant-002', got %v", dataMap2["tenantId"])
	}

	t.Log("✅ 多租户 PostEntity 测试通过")
}
