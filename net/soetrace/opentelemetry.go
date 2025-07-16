package soetrace

// 全新的Tracer 通过oltp协议上报

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.15.0"
	"log"
	"os"
	"time"
)

// OtelTracerConfig 目前仅支持 http上报方式，后续可以兼容起来
type OtelTracerConfig struct {
	Enable        bool    // 是否启用
	ServiceName   string  // 应用名
	Version       string  // 应用版本
	DeploymentEnv string  // 部署环境
	HttpEndpoint  string  // otel 协议上报地址
	HttpUrlPath   string  // otel 协议上报url
	SamplingRatio float64 // 采样比例（例如 1.0 = 全采样，0.1 = 10% 采样）
}

// 设置应用资源
func newResource(ctx context.Context, config OtelTracerConfig) *resource.Resource {
	hostName, _ := os.Hostname()

	r, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),             // 应用名
			semconv.ServiceVersionKey.String(config.Version),              // 应用版本
			semconv.DeploymentEnvironmentKey.String(config.DeploymentEnv), // 部署环境
			semconv.HostNameKey.String(hostName),                          // 主机名
		),
	)

	if err != nil {
		log.Fatalf("%s: %v", "Failed to create OpenTelemetry resource", err)
	}
	return r
}

func newHTTPExporterAndSpanProcessor(ctx context.Context, config OtelTracerConfig) (*otlptrace.Exporter, sdktrace.SpanProcessor) {

	traceExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(config.HttpEndpoint),
		otlptracehttp.WithURLPath(config.HttpUrlPath),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithCompression(1)))

	if err != nil {
		log.Fatalf("%s: %v", "Failed to create the OpenTelemetry trace exporter", err)
	}

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(traceExporter)

	return traceExporter, batchSpanProcessor
}

// InitOpenTelemetry OpenTelemetry 初始化方法
func InitOpenTelemetry(config OtelTracerConfig) func() {
	ctx := context.Background()
	var traceExporter *otlptrace.Exporter
	var batchSpanProcessor sdktrace.SpanProcessor

	traceExporter, batchSpanProcessor = newHTTPExporterAndSpanProcessor(ctx, config)

	otelResource := newResource(ctx, config)

	// 设置采样率
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.SamplingRatio))

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler), //sdktrace.AlwaysSample()
		sdktrace.WithResource(otelResource),
		sdktrace.WithSpanProcessor(batchSpanProcessor))

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExporter.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}
