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
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/soedev/soelib/common/soelog"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/soedev/soelib/common/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	URL             string
	Token           string
	TenantID        string
	ShopCode        string
	TimeoutSecond   int
	Context         context.Context
	CustomClient    *http.Client     // 自定义 HTTP 客户端（可选）
	TransportConfig *TransportConfig // 传输层配置（可选）
	RetryConfig     *RetryConfig     // 重试配置（可选）

	// 熔断配置（实例级，优先级高于全局配置）
	EnableHystrix *bool          // nil=使用全局配置, true=启用, false=禁用
	HystrixConfig *HystrixConfig // 自定义熔断配置（可选，默认使用 DefaultHystrixConfig）

	// OpenTelemetry 链路追踪配置 ⭐
	EnableTracing bool         // 是否启用链路追踪，默认 false
	Tracer        trace.Tracer // 自定义 Tracer，如果为 nil 则使用全局 tracer
}

// TransportConfig 传输层配置
type TransportConfig struct {
	MaxIdleConns        int           // 最大空闲连接数（默认100）
	MaxIdleConnsPerHost int           // 每个主机最大空闲连接数（默认10）
	IdleConnTimeout     time.Duration // 空闲连接超时时间（默认90秒）
	InsecureSkipVerify  bool          // 是否跳过 TLS 证书验证（默认false，生产环境建议false）
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数（默认0，不重试）
	RetryWaitTime   time.Duration // 重试等待时间（默认1秒）
	RetryMaxWait    time.Duration // 最大重试等待时间（默认5秒，指数退避上限）
	RetryableStatus []int         // 可重试的HTTP状态码（默认 []int{500, 502, 503, 504}）
}

// DefaultTransportConfig 返回默认传输层配置
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		InsecureSkipVerify:  false, // 生产环境建议 false
	}
}

// DefaultRetryConfig 返回默认重试配置（不启用重试）
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      0, // 默认不重试
		RetryWaitTime:   time.Second,
		RetryMaxWait:    5 * time.Second,
		RetryableStatus: []int{500, 502, 503, 504},
	}
}

// remoteServiceImpl 服务请求接口实现
type remoteServiceImpl struct {
	url         string
	token       string
	tenantID    string
	shopCode    string
	context     context.Context
	client      *http.Client
	retryConfig *RetryConfig

	// 实例级熔断配置
	enableHystrix *bool  // nil=使用全局配置
	commandName   string // 熔断器命令名（用于隔离不同URL）

	// OpenTelemetry 相关字段 ⭐
	enableTracing bool
	tracer        trace.Tracer
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
	configMu      sync.RWMutex
	alarm         = AlarmConfig{SendErrorToWx: false}
	enableHystrix = false
)

const (
	remotePost = "RemotePost"
	remoteGET  = "RemoteGET"
	remoteDel  = "RemoteDELETE"
)

// InitConfig 初始化http 配置信息（并发安全）
func InitConfig(config SoeHTTPConfig) {
	configMu.Lock()
	defer configMu.Unlock()

	alarm = config.Alarm
	enableHystrix = config.EnableHystrix
	hystrix.ConfigureCommand(remotePost, config.Hystrix)
	hystrix.ConfigureCommand(remoteGET, config.Hystrix)
	hystrix.ConfigureCommand(remoteDel, config.Hystrix)
}

// GetAlarmConfig 获取告警配置（并发安全）
func GetAlarmConfig() AlarmConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return alarm
}

// IsHystrixEnabled 检查熔断是否启用（并发安全）
func IsHystrixEnabled() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return enableHystrix
}

// ResetConfigForTest 重置配置（仅用于测试）
func ResetConfigForTest() {
	configMu.Lock()
	alarm = AlarmConfig{SendErrorToWx: false}
	enableHystrix = false
	configMu.Unlock()

	// 刷新 hystrix 状态和命令配置
	ResetHystrixCommands()
}

var _ SoeRemoteService = (*remoteServiceImpl)(nil)

func NewRemote(opt RemoteOption) SoeRemoteService {
	if opt.Context == nil {
		opt.Context = context.Background()
	}
	if opt.TimeoutSecond <= 0 {
		opt.TimeoutSecond = 15
	}

	// 使用自定义客户端或创建默认客户端
	var client *http.Client
	if opt.CustomClient != nil {
		client = opt.CustomClient
	} else {
		client = createClientWithConfig(opt.TimeoutSecond, opt.TransportConfig)
	}

	// 设置默认重试配置
	retryConfig := opt.RetryConfig
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	// 处理熔断配置
	var commandName string
	if opt.EnableHystrix != nil && *opt.EnableHystrix {
		// 生成唯一的命令名（基于URL）
		commandName = generateCommandName(opt.URL)

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
			tracer = otel.Tracer("soehttp")
		}
	}

	return &remoteServiceImpl{
		url:           opt.URL,
		token:         opt.Token,
		tenantID:      opt.TenantID,
		shopCode:      opt.ShopCode,
		context:       opt.Context,
		client:        client,
		retryConfig:   retryConfig,
		enableHystrix: opt.EnableHystrix,
		commandName:   commandName,
		enableTracing: opt.EnableTracing,
		tracer:        tracer,
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
	soeRemoteService.retryConfig = DefaultRetryConfig() // 设置默认重试配置
	return soeRemoteService
}

func createDefaultClient(timeoutSeconds int) *http.Client {
	return createClientWithConfig(timeoutSeconds, nil)
}

// createClientWithConfig 根据配置创建 HTTP 客户端
func createClientWithConfig(timeoutSeconds int, config *TransportConfig) *http.Client {
	// 使用默认配置或自定义配置
	if config == nil {
		config = DefaultTransportConfig()
		// 为了保持向后兼容性，这里保持原来的行为（跳过证书验证）
		config.InsecureSkipVerify = true
		config.MaxIdleConnsPerHost = 1000
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
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
	// 支持重试时重建 Body
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(*postBody)), nil
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
		return errors.New("未查询到任何信息")
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
		return errors.New("未查询到任何信息")
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
		return errors.New("未查询到任何信息")
	}
	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}
	return nil
}

// shouldUseHystrix 判断是否应该使用熔断
func (s *remoteServiceImpl) shouldUseHystrix() bool {
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
func (s *remoteServiceImpl) do(req *http.Request, operationName string) (result []byte, err error) {
	// 1. 创建 span（如果启用了 tracing）⭐
	var span trace.Span
	ctx := req.Context()
	if s.enableTracing {
		serviceName := extractServiceName(s.url)
		ctx, span = startSpan(ctx, s.tracer, req.Method, serviceName, req.URL.String())
		defer span.End()

		// 记录租户、店铺等信息
		recordSpanAttributes(span, s.tenantID, s.shopCode, s.token)

		// 更新请求的 context
		req = req.WithContext(ctx)
	}

	// 2. 注入 trace context 到 HTTP headers ⭐
	if s.enableTracing {
		injectTraceContext(ctx, req)
	}

	client := s.client
	s.setHeaders(req)

	var statusCode int
	var retryCount int

	// 判断是否使用熔断
	if s.shouldUseHystrix() {
		// 使用实例独立的命令名（如果有），否则使用全局命令名
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

func (s *remoteServiceImpl) doRequest(client *http.Client, req *http.Request) ([]byte, int, int, error) {
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
				soelog.Logger.Info("HTTP 请求重试",
					zap.Int("attempt", attempt),
					zap.Int("maxRetries", maxRetries),
					zap.Duration("waitTime", waitTime),
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
			return nil, 0, attempt, fmt.Errorf("HTTP 请求失败(已重试%d次): %w", maxRetries, err)
		}

		statusCode = resp.StatusCode

		// 读取响应体
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if closeErr != nil && soelog.Logger != nil {
			soelog.Logger.Warn("http do request 关闭流错误", zap.Error(closeErr))
		}

		if readErr != nil {
			lastErr = readErr
			if attempt < maxRetries {
				continue
			}
			return nil, statusCode, attempt, fmt.Errorf("读取响应失败(已重试%d次): %w", maxRetries, readErr)
		}

		// 检查状态码
		if !(resp.StatusCode >= 200 && resp.StatusCode <= 207) {
			// 检查是否是可重试的状态码
			if s.isRetryableStatus(resp.StatusCode) && attempt < maxRetries {
				lastErr = fmt.Errorf("HTTP 状态码 %d", resp.StatusCode)
				continue
			}
			return nil, statusCode, attempt, s.handleErrorWithBody(resp.StatusCode, body)
		}

		// 请求成功
		if attempt > 0 && soelog.Logger != nil {
			soelog.Logger.Info("HTTP 请求重试成功", zap.Int("retryCount", attempt))
		}
		return body, statusCode, attempt, nil
	}

	return nil, statusCode, maxRetries, fmt.Errorf("HTTP 请求失败(已重试%d次): %w", maxRetries, lastErr)
}

// calculateRetryWait 计算重试等待时间（指数退避）
func (s *remoteServiceImpl) calculateRetryWait(attempt int) time.Duration {
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
func (s *remoteServiceImpl) isRetryableStatus(statusCode int) bool {
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

// handleErrorWithBody 使用已读取的响应体处理错误
func (s *remoteServiceImpl) handleErrorWithBody(statusCode int, body []byte) error {
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

func (s *remoteServiceImpl) sendFallbackAlert(req *http.Request, err error) {
	if soelog.Logger != nil {
		soelog.Logger.Info(req.URL.Host + ":" + req.URL.Path + " 熔断降级：" + err.Error())
	}

	// 并发安全地读取告警配置
	configMu.RLock()
	alarmConfig := alarm
	configMu.RUnlock()

	if alarmConfig.SendErrorToWx && alarmConfig.ChatID != "" && alarmConfig.ApiPath != "" {
		content := fmt.Sprintf("请求熔断错误！URL:%s TenantID:%s", s.url, s.tenantID)

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
					soelog.Logger.Warn("发送微信告警超时")
				}
			}
		}()
	}
}

// handleMapError 处理 map 类型的错误响应
func (s *remoteServiceImpl) handleMapError(data map[string]interface{}, statusCode int) error {
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
