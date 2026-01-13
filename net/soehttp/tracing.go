package soehttp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// startSpan 创建一个新的 span 用于 HTTP 请求
// 参数:
//   - ctx: 上下文，可能包含父 span
//   - tracer: OpenTelemetry tracer，如果为 nil 则使用全局 tracer
//   - method: HTTP 方法 (GET, POST, DELETE 等)
//   - serviceName: 目标服务名称
//   - fullURL: 完整的请求 URL
//
// 返回:
//   - 新的 context（包含 span）
//   - span 对象
func startSpan(ctx context.Context, tracer trace.Tracer, method, serviceName, fullURL string) (context.Context, trace.Span) {
	if tracer == nil {
		// 如果没有提供 tracer，使用全局 tracer
		tracer = otel.Tracer("soehttp")
	}

	// Span 名称格式：HTTP {METHOD} {ServiceName}
	spanName := fmt.Sprintf("HTTP %s %s", method, serviceName)

	// 解析 URL 获取路径
	path := extractPath(fullURL)

	// 创建 span
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", fullURL),
			attribute.String("http.target", path),
			attribute.String("service.name", serviceName),
		),
	)

	return ctx, span
}

// recordSpanSuccess 记录成功的请求信息
// 参数:
//   - span: 当前的 span 对象
//   - statusCode: HTTP 响应状态码
//   - responseSize: 响应体大小（字节）
//   - retryCount: 重试次数
func recordSpanSuccess(span trace.Span, statusCode int, responseSize int, retryCount int) {
	if span == nil || !span.IsRecording() {
		return
	}

	span.SetAttributes(
		attribute.Int("http.status_code", statusCode),
		attribute.Int("http.response_size", responseSize),
		attribute.Int("retry.count", retryCount),
	)

	// 根据状态码判断是否成功
	if statusCode >= 200 && statusCode < 400 {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	}
}

// recordSpanError 记录错误信息
// 参数:
//   - span: 当前的 span 对象
//   - err: 错误对象
//   - retryCount: 重试次数
func recordSpanError(span trace.Span, err error, retryCount int) {
	if span == nil || !span.IsRecording() {
		return
	}

	span.SetAttributes(
		attribute.Bool("error", true),
		attribute.Int("retry.count", retryCount),
	)

	// 判断错误类型
	if IsCircuitBreakerError(err) {
		span.SetAttributes(attribute.Bool("circuit_breaker.open", true))
		span.SetStatus(codes.Error, "Circuit breaker triggered")
	} else if IsTimeoutError(err) {
		span.SetStatus(codes.Error, "Request timeout")
	} else if IsNetworkError(err) {
		span.SetStatus(codes.Error, "Network error")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}

	// 记录错误详情
	span.RecordError(err)
}

// recordSpanAttributes 记录额外的属性（租户、店铺等）
// 参数:
//   - span: 当前的 span 对象
//   - tenantID: 租户 ID
//   - shopCode: 店铺代码
//   - token: 认证 token
func recordSpanAttributes(span trace.Span, tenantID, shopCode, token string) {
	if span == nil || !span.IsRecording() {
		return
	}

	if tenantID != "" {
		span.SetAttributes(attribute.String("tenant.id", tenantID))
	}
	if shopCode != "" {
		span.SetAttributes(attribute.String("shop.code", shopCode))
	}
	if token != "" {
		// 只记录 token 的前 10 个字符，避免泄露敏感信息
		if len(token) > 10 {
			span.SetAttributes(attribute.String("auth.token_prefix", token[:10]+"..."))
		} else {
			span.SetAttributes(attribute.String("auth.token_prefix", token+"..."))
		}
	}
}

// injectTraceContext 将 trace context 注入到 HTTP headers
// 这样下游服务就能继续追踪同一个 trace
// 参数:
//   - ctx: 包含 span 的上下文
//   - req: HTTP 请求对象
func injectTraceContext(ctx context.Context, req *http.Request) {
	// 使用全局的 propagator（在 soetrace.InitOpenTelemetry 中设置）
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// extractPath 从完整 URL 中提取路径部分
// 例如: http://localhost:8080/api/users?id=123 -> /api/users
func extractPath(fullURL string) string {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}
	return parsedURL.Path
}

// extractServiceName 从 URL 中提取服务名
// 例如: http://user-service:8080/api/users -> user-service
// 例如: http://localhost:8080/api/users -> localhost
func extractServiceName(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "unknown"
	}

	host := parsedURL.Host
	// 移除端口号
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host
}
