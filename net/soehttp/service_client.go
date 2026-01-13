package soehttp

/**
  服务级 HTTP 客户端（支持连接池复用）
  适用于微服务架构中，需要频繁调用同一个服务的多个接口的场景
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
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/mitchellh/mapstructure"
	"github.com/soedev/soelib/common/soelog"
	"github.com/soedev/soelib/common/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ServiceClientOption 服务客户端配置
type ServiceClientOption struct {
	ServiceName     string           // 服务名称（用于标识和日志）
	BaseURL         string           // 服务基础 URL（如：http://worker-service）
	TimeoutSecond   int              // 超时时间（秒）
	Context         context.Context  // 上下文
	TransportConfig *TransportConfig // 传输层配置（可选）
	RetryConfig     *RetryConfig     // 重试配置（可选）
	// 熔断配置
	EnableHystrix *bool          // nil=使用全局配置, true=启用, false=禁用
	HystrixConfig *HystrixConfig // 自定义熔断配置（可选）
	// OpenTelemetry 链路追踪配置 ⭐
	EnableTracing bool         // 是否启用链路追踪，默认 false
	Tracer        trace.Tracer // 自定义 Tracer，如果为 nil 则使用全局 tracer
	TracerName    string       // Tracer 名称，默认 "soehttp"
}

// RequestOptions 请求级选项（可选参数）
type RequestOptions struct {
	TenantID string // 租户 ID（请求级，优先于客户端级）
	ShopCode string // 商户代码（请求级，优先于客户端级）
	Token    string // 认证 Token（请求级，优先于客户端级）
}

// SoeServiceClient 服务客户端接口
type SoeServiceClient interface {
	// Post 发送 POST 请求（相对路径，如：/api/worker/get）
	Post(path string, postBody *[]byte) ([]byte, error)
	// Get 发送 GET 请求（相对路径）
	Get(path string, reader io.Reader) ([]byte, error)
	// Delete 发送 DELETE 请求（相对路径）
	Delete(path string, reader io.Reader) ([]byte, error)
	// PostEntity 发送 POST 请求并自动解析响应
	PostEntity(path string, v interface{}, r interface{}) error
	// GetEntity 发送 GET 请求并自动解析响应
	GetEntity(path string, v interface{}) error
	// DeleteEntity 发送 DELETE 请求并自动解析响应
	DeleteEntity(path string, v interface{}) error

	// PostWithOptions 发送 POST 请求（支持请求级选项）⭐ 新增
	PostWithOptions(path string, postBody *[]byte, opts RequestOptions) ([]byte, error)
	// GetWithOptions 发送 GET 请求（支持请求级选项）⭐ 新增
	GetWithOptions(path string, reader io.Reader, opts RequestOptions) ([]byte, error)
	// DeleteWithOptions 发送 DELETE 请求（支持请求级选项）⭐ 新增
	DeleteWithOptions(path string, reader io.Reader, opts RequestOptions) ([]byte, error)
	// PostEntityWithOptions 发送 POST 请求并自动解析响应（支持请求级选项）⭐ 新增
	PostEntityWithOptions(path string, v interface{}, r interface{}, opts RequestOptions) error
	// GetEntityWithOptions 发送 GET 请求并自动解析响应（支持请求级选项）⭐ 新增
	GetEntityWithOptions(path string, v interface{}, opts RequestOptions) error
	// DeleteEntityWithOptions 发送 DELETE 请求并自动解析响应（支持请求级选项）⭐ 新增
	DeleteEntityWithOptions(path string, v interface{}, opts RequestOptions) error

	// GetServiceName 获取服务名称
	GetServiceName() string
	// GetBaseURL 获取基础 URL
	GetBaseURL() string
}

// serviceClientImpl 服务客户端实现
type serviceClientImpl struct {
	serviceName   string
	baseURL       string
	context       context.Context
	client        *http.Client // 共享的 HTTP Client（关键！）
	retryConfig   *RetryConfig
	enableHystrix *bool
	commandName   string
	// OpenTelemetry 相关字段 ⭐
	enableTracing bool
	tracer        trace.Tracer
}

// serviceClientPool 服务客户端池（全局管理）
type serviceClientPool struct {
	clients map[string]SoeServiceClient
	mu      sync.RWMutex
}

var (
	globalServicePool     *serviceClientPool
	globalServicePoolOnce sync.Once
)

// getServiceClientPool 获取全局服务客户端池
func getServiceClientPool() *serviceClientPool {
	globalServicePoolOnce.Do(func() {
		globalServicePool = &serviceClientPool{
			clients: make(map[string]SoeServiceClient),
		}
	})
	return globalServicePool
}

// NewServiceClient 创建服务客户端（推荐用法）
// 为同一个服务的所有接口提供共享的连接池
func NewServiceClient(opt ServiceClientOption) SoeServiceClient {
	if opt.Context == nil {
		opt.Context = context.Background()
	}
	if opt.TimeoutSecond <= 0 {
		opt.TimeoutSecond = 15
	}
	if opt.ServiceName == "" {
		opt.ServiceName = "unknown-service"
	}

	// 创建共享的 HTTP Client（关键：连接池在这里）
	client := createClientWithConfig(opt.TimeoutSecond, opt.TransportConfig)

	// 设置默认重试配置
	retryConfig := opt.RetryConfig
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	// 处理熔断配置
	var commandName string
	if opt.EnableHystrix != nil && *opt.EnableHystrix {
		// 使用服务名作为熔断器命令名（同一服务共享熔断器）
		commandName = "service-" + opt.ServiceName

		// 配置熔断器
		hystrixCfg := opt.HystrixConfig
		if hystrixCfg == nil {
			hystrixCfg = DefaultHystrixConfig()
		}
		configureHystrixCommand(commandName, hystrixCfg)
	}

	// 初始化 OpenTelemetry Tracer ⭐
	var tracer trace.Tracer
	if opt.EnableTracing {
		if opt.Tracer != nil {
			tracer = opt.Tracer
		} else {
			// 使用全局 tracer
			tracerName := opt.TracerName
			if tracerName == "" {
				tracerName = "soehttp"
			}
			tracer = otel.Tracer(tracerName)
		}
	}

	return &serviceClientImpl{
		serviceName:   opt.ServiceName,
		baseURL:       opt.BaseURL,
		context:       opt.Context,
		client:        client,
		retryConfig:   retryConfig,
		enableHystrix: opt.EnableHystrix,
		commandName:   commandName,
		enableTracing: opt.EnableTracing,
		tracer:        tracer,
	}
}

// RegisterService 注册服务客户端到全局池（可选用法）
// 适用于在应用启动时集中注册所有服务
func RegisterService(serviceName string, opt ServiceClientOption) {
	opt.ServiceName = serviceName
	client := NewServiceClient(opt)

	pool := getServiceClientPool()
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.clients[serviceName] = client
}

// GetServiceClient 从全局池获取服务客户端（配合 RegisterService 使用）
func GetServiceClient(serviceName string) (SoeServiceClient, error) {
	pool := getServiceClientPool()
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	client, ok := pool.clients[serviceName]
	if !ok {
		return nil, errors.New("服务客户端未注册: " + serviceName)
	}
	return client, nil
}

// GetServiceName 获取服务名称
func (s *serviceClientImpl) GetServiceName() string {
	return s.serviceName
}

// GetBaseURL 获取基础 URL
func (s *serviceClientImpl) GetBaseURL() string {
	return s.baseURL
}

// Post 发送 POST 请求
func (s *serviceClientImpl) Post(path string, postBody *[]byte) ([]byte, error) {
	return s.PostWithOptions(path, postBody, RequestOptions{})
}

// PostWithOptions 发送 POST 请求（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) PostWithOptions(path string, postBody *[]byte, opts RequestOptions) ([]byte, error) {
	fullURL := s.baseURL + path
	req, err := http.NewRequestWithContext(s.context, "POST", fullURL, bytes.NewReader(*postBody))
	if err != nil {
		return nil, err
	}

	// 支持重试时重建 Body
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(*postBody)), nil
	}

	return s.doWithOptions(req, "RemotePost", opts)
}

// Get 发送 GET 请求
func (s *serviceClientImpl) Get(path string, reader io.Reader) ([]byte, error) {
	return s.GetWithOptions(path, reader, RequestOptions{})
}

// GetWithOptions 发送 GET 请求（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) GetWithOptions(path string, reader io.Reader, opts RequestOptions) ([]byte, error) {
	fullURL := s.baseURL + path
	req, err := http.NewRequestWithContext(s.context, "GET", fullURL, reader)
	if err != nil {
		return nil, err
	}
	return s.doWithOptions(req, "RemoteGET", opts)
}

// Delete 发送 DELETE 请求
func (s *serviceClientImpl) Delete(path string, reader io.Reader) ([]byte, error) {
	return s.DeleteWithOptions(path, reader, RequestOptions{})
}

// DeleteWithOptions 发送 DELETE 请求（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) DeleteWithOptions(path string, reader io.Reader, opts RequestOptions) ([]byte, error) {
	fullURL := s.baseURL + path
	req, err := http.NewRequestWithContext(s.context, "DELETE", fullURL, reader)
	if err != nil {
		return nil, err
	}
	return s.doWithOptions(req, "RemoteDELETE", opts)
}

// PostEntity 发送 POST 请求并自动解析响应
func (s *serviceClientImpl) PostEntity(path string, v interface{}, r interface{}) error {
	return s.PostEntityWithOptions(path, v, r, RequestOptions{})
}

// PostEntityWithOptions 发送 POST 请求并自动解析响应（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) PostEntityWithOptions(path string, v interface{}, r interface{}, opts RequestOptions) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body, err := s.PostWithOptions(path, &b, opts)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("未查询到任何信息")
	}
	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}
	return nil
}

// GetEntity 发送 GET 请求并自动解析响应
func (s *serviceClientImpl) GetEntity(path string, v interface{}) error {
	return s.GetEntityWithOptions(path, v, RequestOptions{})
}

// GetEntityWithOptions 发送 GET 请求并自动解析响应（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) GetEntityWithOptions(path string, v interface{}, opts RequestOptions) error {
	body, err := s.GetWithOptions(path, nil, opts)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("未查询到任何信息")
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

// DeleteEntity 发送 DELETE 请求并自动解析响应
func (s *serviceClientImpl) DeleteEntity(path string, v interface{}) error {
	return s.DeleteEntityWithOptions(path, v, RequestOptions{})
}

// DeleteEntityWithOptions 发送 DELETE 请求并自动解析响应（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) DeleteEntityWithOptions(path string, v interface{}, opts RequestOptions) error {
	body, err := s.DeleteWithOptions(path, nil, opts)
	if err != nil {
		return err
	}
	if body == nil {
		return errors.New("未查询到任何信息")
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}

// shouldUseHystrix 判断是否应该使用熔断
func (s *serviceClientImpl) shouldUseHystrix() bool {
	// 优先级1: 实例级配置
	if s.enableHystrix != nil {
		return *s.enableHystrix
	}

	// 优先级2: 全局配置
	configMu.RLock()
	globalEnabled := enableHystrix
	configMu.RUnlock()

	return globalEnabled
}

// do 执行 HTTP 请求（支持可选的熔断）
func (s *serviceClientImpl) do(req *http.Request, operationName string) (result []byte, err error) {
	return s.doWithOptions(req, operationName, RequestOptions{})
}

// doWithOptions 执行 HTTP 请求（支持请求级选项）⭐ 新增
func (s *serviceClientImpl) doWithOptions(req *http.Request, operationName string, opts RequestOptions) (result []byte, err error) {
	// 1. 创建 span（如果启用了 tracing）⭐
	var span trace.Span
	ctx := req.Context()
	if s.enableTracing {
		ctx, span = startSpan(ctx, s.tracer, req.Method, s.serviceName, req.URL.String())
		defer span.End()

		// 记录租户、店铺等信息
		recordSpanAttributes(span, opts.TenantID, opts.ShopCode, opts.Token)

		// 更新请求的 context
		req = req.WithContext(ctx)
	}

	// 2. 注入 trace context 到 HTTP headers ⭐
	if s.enableTracing {
		injectTraceContext(ctx, req)
	}

	client := s.client
	s.setHeadersWithOptions(req, opts)

	var statusCode int
	var retryCount int

	// 判断是否使用熔断
	if s.shouldUseHystrix() {
		// 使用服务级的命令名（如果有），否则使用全局命令名
		cmdName := s.commandName
		if cmdName == "" {
			// 使用全局命令名（按请求类型）
			cmdName = operationName
		}

		// 启用熔断
		err = hystrix.Do(cmdName, func() error {
			result, statusCode, retryCount, err = s.doRequest(client, req)
			return err
		}, func(e error) error {
			// 发生熔断处理逻辑
			s.sendFallbackAlert(req, e)
			return errors.New("fallback")
		})
	} else {
		// 不使用熔断，直接执行
		result, statusCode, retryCount, err = s.doRequest(client, req)
	}

	// 3. 记录 span 信息 ⭐
	if span != nil {
		if err != nil {
			recordSpanError(span, err, retryCount)
		} else {
			recordSpanSuccess(span, statusCode, len(result), retryCount)
		}
	}

	return result, err
}

// doRequest 执行实际的 HTTP 请求（与 remoteServiceImpl 共享逻辑）
func (s *serviceClientImpl) doRequest(client *http.Client, req *http.Request) ([]byte, int, int, error) {
	maxRetries := 0
	if s.retryConfig != nil {
		maxRetries = s.retryConfig.MaxRetries
	}

	var lastErr error
	var statusCode int
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 如果是重试，等待一段时间
		if attempt > 0 {
			waitTime := s.calculateRetryWait(attempt)
			if soelog.Logger != nil {
				soelog.Logger.Info("ServiceClient HTTP 请求重试",
					zap.Int("attempt", attempt),
					zap.Int("maxRetries", maxRetries),
					zap.Duration("waitTime", waitTime),
					zap.String("service", s.serviceName),
					zap.String("url", req.URL.String()))
			}
			time.Sleep(waitTime)
		}

		// 执行请求
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// 网络错误，如果还有重试次数，继续重试
			if attempt < maxRetries {
				continue
			}
			return nil, 0, attempt, err
		}

		statusCode = resp.StatusCode

		// 读取响应体
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if closeErr != nil && soelog.Logger != nil {
			soelog.Logger.Warn("ServiceClient 关闭响应流错误",
				zap.Error(closeErr),
				zap.String("service", s.serviceName))
		}

		if readErr != nil {
			lastErr = readErr
			if attempt < maxRetries {
				continue
			}
			return nil, statusCode, attempt, readErr
		}

		// 检查状态码
		if !(resp.StatusCode >= 200 && resp.StatusCode <= 207) {
			// 检查是否是可重试的状态码
			if s.isRetryableStatus(resp.StatusCode) && attempt < maxRetries {
				lastErr = errors.New("HTTP 状态码: " + string(rune(resp.StatusCode)))
				continue
			}
			return nil, statusCode, attempt, s.handleErrorWithBody(resp.StatusCode, body)
		}

		// 请求成功
		if attempt > 0 && soelog.Logger != nil {
			soelog.Logger.Info("ServiceClient HTTP 请求重试成功",
				zap.Int("retryCount", attempt),
				zap.String("service", s.serviceName))
		}
		return body, statusCode, attempt, nil
	}

	return nil, statusCode, maxRetries, lastErr
}

// calculateRetryWait 计算重试等待时间（指数退避）
func (s *serviceClientImpl) calculateRetryWait(attempt int) time.Duration {
	if s.retryConfig == nil {
		return time.Second
	}

	// 指数退避: baseWait * 2^(attempt-1)
	waitTime := s.retryConfig.RetryWaitTime * time.Duration(1<<uint(attempt-1))

	// 限制最大等待时间
	if waitTime > s.retryConfig.RetryMaxWait {
		waitTime = s.retryConfig.RetryMaxWait
	}

	return waitTime
}

// isRetryableStatus 检查状态码是否可重试
func (s *serviceClientImpl) isRetryableStatus(statusCode int) bool {
	if s.retryConfig == nil || len(s.retryConfig.RetryableStatus) == 0 {
		// 默认可重试的状态码
		return statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
	}

	for _, code := range s.retryConfig.RetryableStatus {
		if statusCode == code {
			return true
		}
	}
	return false
}

// handleErrorWithBody 使用已读取的响应体处理错误（与 remoteServiceImpl 共享逻辑）
func (s *serviceClientImpl) handleErrorWithBody(statusCode int, body []byte) error {
	// 处理 401/403 认证错误
	if statusCode == 401 || statusCode == 403 {
		return errors.New(http.StatusText(statusCode))
	}

	// 尝试解析 JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// 如果不是 JSON，返回状态码文本
		return errors.New(http.StatusText(statusCode))
	}

	// 根据数据类型处理
	switch t := data.(type) {
	case string:
		return errors.New(t)
	case map[string]interface{}:
		return s.handleMapError(t, statusCode)
	default:
		return errors.New("服务器太忙了，请稍后再试！")
	}
}

// handleMapError 处理 map 类型的错误响应（与 remoteServiceImpl 共享逻辑）
func (s *serviceClientImpl) handleMapError(data map[string]interface{}, statusCode int) error {
	// 尝试解析为 SoeRestAPIException
	var soeRestAPIException SoeRestAPIException
	if err := mapstructure.Decode(data, &soeRestAPIException); err == nil {
		// Data 字段优先
		if soeRestAPIException.Data != "" {
			return errors.New(soeRestAPIException.Data)
		}
		// Message 字段次之
		if soeRestAPIException.Message != "" {
			return errors.New(soeRestAPIException.Message)
		}
	}

	// 尝试解析为 SoeGoResponseVO
	var soeGoResponseVO SoeGoResponseVO
	if err := mapstructure.Decode(data, &soeGoResponseVO); err == nil {
		if soeGoResponseVO.Code == 500 {
			// 安全地处理 Data 字段
			if dataStr, ok := soeGoResponseVO.Data.(string); ok && dataStr != "" {
				return errors.New(dataStr)
			}
		}
		// 返回 Msg 字段
		if soeGoResponseVO.Msg != "" {
			return errors.New(soeGoResponseVO.Msg)
		}
	}

	// 默认错误信息
	return errors.New("服务器太忙了，请稍后再试！")
}

// sendFallbackAlert 发送熔断告警
func (s *serviceClientImpl) sendFallbackAlert(req *http.Request, err error) {
	if soelog.Logger != nil {
		soelog.Logger.Info(fmt.Sprintf("ServiceClient 熔断降级，Service: %s, URL: %s, Error: %s",
			s.serviceName, req.URL.String(), err.Error()))
	}

	// 并发安全地读取告警配置
	configMu.RLock()
	alarmConfig := alarm
	configMu.RUnlock()

	if alarmConfig.SendErrorToWx && alarmConfig.ChatID != "" && alarmConfig.ApiPath != "" {
		content := "ServiceClient 熔断错误！Service:" + s.serviceName +
			" URL:" + req.URL.String()

		// 使用带超时的 goroutine 防止泄漏
		go func() {
			done := make(chan struct{})
			go func() {
				utils.SendMsgToWorkWx(alarmConfig.ChatID, content, alarmConfig.ApiPath, utils.WorkWxRestTokenStr)
				close(done)
			}()

			select {
			case <-done:
				// 发送成功
			case <-time.After(5 * time.Second):
				// 超时保护
				if soelog.Logger != nil {
					soelog.Logger.Warn(fmt.Sprintf("ServiceClient 发送微信告警超时，Service: %s", s.serviceName))
				}
			}
		}()
	}
}

// setHeaders 设置请求头（向后兼容）
func (s *serviceClientImpl) setHeaders(req *http.Request) {
	s.setHeadersWithOptions(req, RequestOptions{})
}

// setHeadersWithOptions 设置请求头（支持请求级选项）⭐ 新增
// 优先级：请求级参数 > 客户端级参数
func (s *serviceClientImpl) setHeadersWithOptions(req *http.Request, opts RequestOptions) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Token：请求级优先
	token := opts.Token
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	// TenantID：请求级优先
	tenantID := opts.TenantID
	if tenantID != "" {
		req.Header.Set("tenantId", tenantID)
	}

	// ShopCode：请求级优先
	shopCode := opts.ShopCode
	if shopCode != "" {
		req.Header.Set("shopCode", shopCode)
	}
}

// createDefaultHTTPClient 创建默认的 HTTP Client（便捷方法）
func createDefaultHTTPClient(timeoutSeconds int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20, // 服务级连接池建议值
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
