package soetrace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type registeredTracer struct {
	isRegistered bool
}

var (
	globalTracer = registeredTracer{false}
)

// IsGlobalTracerRegistered returns a `bool` to indicate if a tracer has been globally registered
func IsGlobalTracerRegistered() bool {
	return globalTracer.isRegistered
}

// ExtractTraceID 获取当前上下文中链路ID
func ExtractTraceID(ctx context.Context) string {
	if !IsGlobalTracerRegistered() {
		return ""
	}
	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		return span.TraceID().String()
	}
	return ""
}

// OtelTracer 索易自定义链路信息
type OtelTracer struct {
	tracer trace.Tracer
	kind   trace.SpanKind
	opt    *options
}

// NewTracer 创建Tracer
func NewTracer(kind trace.SpanKind, opts ...Option) *OtelTracer {
	op := options{
		//	propagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}), ego 写法
		propagator: otel.GetTextMapPropagator(), // 新写法
	}
	for _, o := range opts {
		o(&op)
	}
	return &OtelTracer{tracer: otel.Tracer("soe"), kind: kind, opt: &op}
}

// Start 开始链路记录： span记录当前步骤参数以及状态
func (t *OtelTracer) Start(ctx context.Context, operation string, carrier propagation.TextMapCarrier, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if (t.kind == trace.SpanKindServer || t.kind == trace.SpanKindConsumer) && carrier != nil {
		ctx = t.opt.propagator.Extract(ctx, carrier)
	}
	opts = append(opts, trace.WithSpanKind(t.kind))

	ctx, span := t.tracer.Start(ctx, operation, opts...)

	// 服务之间调用： 链路对接处理
	if (t.kind == trace.SpanKindClient || t.kind == trace.SpanKindProducer) && carrier != nil {
		t.opt.propagator.Inject(ctx, carrier)
	}
	return ctx, span
}

type options struct {
	propagator propagation.TextMapPropagator
}

// Option is tracing option.
type Option func(*options)

// CustomTag 自定义属性方法
func CustomTag(key string, val string) attribute.KeyValue {
	return attribute.String(key, val)
}
