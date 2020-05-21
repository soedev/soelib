package soetrace

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func SetUpUseJaeger(config JaegerTracerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.JaegerOpen {
			var parentSpan opentracing.Span
			// tracer, closer := NewJaegerTracer(config)
			// defer closer.Close()
			spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
			if err == nil {
				parentSpan = opentracing.StartSpan(c.Request.URL.Path)
				defer parentSpan.Finish()
			} else {
				parentSpan = opentracing.StartSpan(
					c.Request.URL.Path,
					opentracing.ChildOf(spCtx),
					opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
					ext.SpanKindRPCServer,
				)
				defer parentSpan.Finish()
			}
			c.Set("Tracer", opentracing.GlobalTracer())
			c.Set("ParentSpanContext", parentSpan.Context())
		}
		c.Next()
	}
}
