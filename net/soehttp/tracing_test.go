package soehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// setupTestTracer 设置测试用的 tracer
func setupTestTracer() (*tracetest.SpanRecorder, trace.Tracer) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer("test")
	return spanRecorder, tracer
}

// TestServiceClient_TracingEnabled 测试 ServiceClient 启用链路追踪
func TestServiceClient_TracingEnabled(t *testing.T) {
	// 设置测试 tracer
	spanRecorder, tracer := setupTestTracer()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 trace context 已注入
		traceParent := r.Header.Get("traceparent")
		assert.NotEmpty(t, traceParent, "trace context 应该被注入到 headers 中")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	// 创建带 tracing 的 ServiceClient
	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		TimeoutSecond: 5,
		EnableTracing: true,
		Tracer:        tracer,
	})

	// 发送请求
	_, err := client.Get("/test", nil)
	assert.NoError(t, err)

	// 验证 span 被创建
	spans := spanRecorder.Ended()
	assert.Equal(t, 1, len(spans), "应该创建 1 个 span")

	// 验证 span 属性
	span := spans[0]
	assert.Equal(t, "HTTP GET test-service", span.Name())
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())

	// 验证 span 属性
	attrs := span.Attributes()
	assertAttribute(t, attrs, "http.method", "GET")
	assertAttribute(t, attrs, "service.name", "test-service")
	assertAttribute(t, attrs, "http.status_code", 200)
}

// TestServiceClient_TracingWithOptions 测试带请求级选项的链路追踪
func TestServiceClient_TracingWithOptions(t *testing.T) {
	spanRecorder, tracer := setupTestTracer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		EnableTracing: true,
		Tracer:        tracer,
	})

	// 使用 WithOptions 发送请求
	_, err := client.GetWithOptions("/test", nil, RequestOptions{
		TenantID: "TENANT-123",
		ShopCode: "SHOP-456",
		Token:    "test-token-123",
	})
	assert.NoError(t, err)

	// 验证 span 属性包含租户信息
	spans := spanRecorder.Ended()
	assert.Equal(t, 1, len(spans))

	attrs := spans[0].Attributes()
	assertAttribute(t, attrs, "tenant.id", "TENANT-123")
	assertAttribute(t, attrs, "shop.code", "SHOP-456")
	// token 应该被截断
	assertAttributeContains(t, attrs, "auth.token_prefix", "test-token")
}

// TestServiceClient_TracingWithRetry 测试重试场景的链路追踪
func TestServiceClient_TracingWithRetry(t *testing.T) {
	spanRecorder, tracer := setupTestTracer()

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			// 第一次请求失败
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// 第二次请求成功
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		}
	}))
	defer server.Close()

	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		EnableTracing: true,
		Tracer:        tracer,
		RetryConfig: &RetryConfig{
			MaxRetries: 2,
		},
	})

	_, err := client.Get("/test", nil)
	assert.NoError(t, err)

	// 验证 span 记录了重试次数
	spans := spanRecorder.Ended()
	assert.Equal(t, 1, len(spans))

	attrs := spans[0].Attributes()
	assertAttribute(t, attrs, "retry.count", 1) // 重试了 1 次
}

// TestServiceClient_TracingWithError 测试错误场景的链路追踪
func TestServiceClient_TracingWithError(t *testing.T) {
	spanRecorder, tracer := setupTestTracer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		EnableTracing: true,
		Tracer:        tracer,
		RetryConfig: &RetryConfig{
			MaxRetries: 0, // 不重试
		},
	})

	_, err := client.Get("/test", nil)
	assert.Error(t, err)

	// 验证 span 记录了错误
	spans := spanRecorder.Ended()
	assert.Equal(t, 1, len(spans))

	span := spans[0]
	
	// 验证 span 状态为 Error
	assert.Equal(t, "Error", span.Status().Code.String())
	
	// 验证有 error 属性
	attrs := span.Attributes()
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "error" {
			found = true
			assert.True(t, attr.Value.AsBool())
			break
		}
	}
	assert.True(t, found, "应该有 error 属性")
}

// TestServiceClient_TracingDisabled 测试禁用链路追踪
func TestServiceClient_TracingDisabled(t *testing.T) {
	spanRecorder, _ := setupTestTracer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证没有注入 trace context
		traceParent := r.Header.Get("traceparent")
		assert.Empty(t, traceParent, "禁用 tracing 时不应该注入 trace context")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	// 创建不启用 tracing 的 ServiceClient
	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		EnableTracing: false, // 禁用
	})

	_, err := client.Get("/test", nil)
	assert.NoError(t, err)

	// 验证没有创建 span
	spans := spanRecorder.Ended()
	assert.Equal(t, 0, len(spans), "禁用 tracing 时不应该创建 span")
}

// TestRemoteService_TracingEnabled 测试 RemoteService（传统 API）的链路追踪
func TestRemoteService_TracingEnabled(t *testing.T) {
	spanRecorder, tracer := setupTestTracer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 trace context 已注入
		traceParent := r.Header.Get("traceparent")
		assert.NotEmpty(t, traceParent)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	// 创建带 tracing 的 RemoteService
	remote := NewRemote(RemoteOption{
		URL:           server.URL + "/test",
		TimeoutSecond: 5,
		EnableTracing: true,
		Tracer:        tracer,
	})

	_, err := remote.Get(nil)
	assert.NoError(t, err)

	// 验证 span 被创建
	spans := spanRecorder.Ended()
	assert.Equal(t, 1, len(spans))

	span := spans[0]
	assert.Contains(t, span.Name(), "HTTP GET")
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())
}

// TestRemoteService_TracingDisabled 测试 RemoteService 禁用链路追踪
func TestRemoteService_TracingDisabled(t *testing.T) {
	spanRecorder, _ := setupTestTracer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	// 不启用 tracing
	remote := NewRemote(RemoteOption{
		URL:           server.URL + "/test",
		EnableTracing: false,
	})

	_, err := remote.Get(nil)
	assert.NoError(t, err)

	// 验证没有创建 span
	spans := spanRecorder.Ended()
	assert.Equal(t, 0, len(spans))
}

// TestTraceContextPropagation 测试 trace context 传播
func TestTraceContextPropagation(t *testing.T) {
	spanRecorder, tracer := setupTestTracer()

	// 创建父 span
	ctx, parentSpan := tracer.Start(context.Background(), "parent-operation")
	defer parentSpan.End()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 trace context 传播
		traceParent := r.Header.Get("traceparent")
		assert.NotEmpty(t, traceParent)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	client := NewServiceClient(ServiceClientOption{
		ServiceName:   "test-service",
		BaseURL:       server.URL,
		Context:       ctx, // 使用带有父 span 的 context
		EnableTracing: true,
		Tracer:        tracer,
	})

	_, err := client.Get("/test", nil)
	assert.NoError(t, err)

	// 验证创建了 2 个 span（父 + 子）
	spans := spanRecorder.Ended()
	assert.GreaterOrEqual(t, len(spans), 1)

	// 找到 HTTP span
	var httpSpan *sdktrace.ReadOnlySpan
	for i := range spans {
		if spans[i].Name() == "HTTP GET test-service" {
			span := spans[i]
			httpSpan = &span
			break
		}
	}
	assert.NotNil(t, httpSpan, "应该找到 HTTP span")

	// 验证 HTTP span 的父 span 是 parent-operation
	parentSC := parentSpan.SpanContext()
	httpSC := (*httpSpan).SpanContext()
	assert.Equal(t, parentSC.TraceID(), httpSC.TraceID(), "应该在同一个 trace 中")
}

// 辅助函数：验证 span 属性
func assertAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expectedValue interface{}) {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			switch v := expectedValue.(type) {
			case string:
				assert.Equal(t, v, attr.Value.AsString())
			case int:
				assert.Equal(t, int64(v), attr.Value.AsInt64())
			case int64:
				assert.Equal(t, v, attr.Value.AsInt64())
			}
			return
		}
	}
	t.Errorf("未找到属性: %s", key)
}

// 辅助函数：验证 span 属性包含字符串
func assertAttributeContains(t *testing.T, attrs []attribute.KeyValue, key string, contains string) {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			assert.Contains(t, attr.Value.AsString(), contains)
			return
		}
	}
	t.Errorf("未找到属性: %s", key)
}
