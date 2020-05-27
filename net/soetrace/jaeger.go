package soetrace

import (
	"context"
	"fmt"
	"io"
	infoLog "log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type JaegerTracerConfig struct {
	JaegerOpen   bool
	UseAliTracer bool
	Config       tracerConfig
}

type tracerConfig struct {
	ServiceName string //服务名称
	LogSpans    bool
	Endpoint    string               //jaeger host url
	Sampler     config.SamplerConfig //采样参数配置
	InitSpans   []string
}

type spanTracer struct {
	Tracer opentracing.Tracer //服务名称
	Closer io.Closer
	Enable bool //初始化状态
}

var serverTracer = spanTracer{Enable: false}

//默认的四种 DBTracer
var dbMongoTracer = spanTracer{Enable: false}
var dbMssqlTracer = spanTracer{Enable: false}
var dbPgTracer = spanTracer{Enable: false}
var dbRedisTracer = spanTracer{Enable: false}

const (
	spanDBMongo = "mongo"
	spanDBMSSQL = "mssql"
	spanDBPG    = "postgres"
	spanDBREDIS = "redis"
)

type MDReaderWriter struct {
	metadata.MD
}

func NewJaegerTracer(jtconfig JaegerTracerConfig) bool {
	if jtconfig.JaegerOpen {
		serverTracer.Tracer, serverTracer.Closer = initJaegerTracer(jtconfig.Config.ServiceName, jtconfig)
		opentracing.SetGlobalTracer(serverTracer.Tracer)
		initSpanTracer(jtconfig)
		return true
	}
	return false
}

func CloseTracer() {
	if serverTracer.Enable {
		serverTracer.Closer.Close()
	}
	if dbMongoTracer.Enable {
		dbMongoTracer.Closer.Close()
	}
	if dbMssqlTracer.Enable {
		dbMssqlTracer.Closer.Close()
	}
	if dbPgTracer.Enable {
		dbPgTracer.Closer.Close()
	}
	if dbRedisTracer.Enable {
		dbRedisTracer.Closer.Close()
	}
}

func initJaegerTracer(serviceName string, jtconfig JaegerTracerConfig) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler:     &jtconfig.Config.Sampler,
		ServiceName: serviceName,
	}
	if jtconfig.UseAliTracer {
		cfg.Reporter = &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: jtconfig.Config.Endpoint,
		}
	} else {
		cfg.Reporter = &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: jtconfig.Config.Endpoint,
		}
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func initSpanTracer(jtconfig JaegerTracerConfig) {
	initSpans := ""
	for i := 0; i < len(jtconfig.Config.InitSpans); i++ {
		if jtconfig.Config.InitSpans[i] == spanDBMongo && !dbMongoTracer.Enable {
			dbMongoTracer.Tracer, dbMongoTracer.Closer = initJaegerTracer(spanDBMongo, jtconfig)
			dbMongoTracer.Enable = true
			initSpans = initSpans + spanDBMongo + " "
		} else if jtconfig.Config.InitSpans[i] == spanDBMSSQL && !dbMssqlTracer.Enable {
			dbMssqlTracer.Tracer, dbMssqlTracer.Closer = initJaegerTracer(spanDBMSSQL, jtconfig)
			dbMssqlTracer.Enable = true
			initSpans = initSpans + spanDBMSSQL + " "
		} else if jtconfig.Config.InitSpans[i] == spanDBPG && !dbPgTracer.Enable {
			dbPgTracer.Tracer, dbPgTracer.Closer = initJaegerTracer(spanDBPG, jtconfig)
			dbPgTracer.Enable = true
			initSpans = initSpans + spanDBPG + " "
		} else if jtconfig.Config.InitSpans[i] == spanDBREDIS && !dbRedisTracer.Enable {
			dbRedisTracer.Tracer, dbRedisTracer.Closer = initJaegerTracer(spanDBREDIS, jtconfig)
			dbRedisTracer.Enable = true
			initSpans = initSpans + spanDBREDIS + " "
		}
	}
	if initSpans != "" {
		infoLog.Println(fmt.Sprintf("%s Tracer 已经初始化", initSpans))
	}
}

//GetNewSpanFromContext 获取新的Span用来记录
func GetNewSpanFromContext(c *gin.Context, operationName string) (opentracing.Span, bool) {
	if c != nil {
		// tracer, isExists1 := c.Get("Tracer")
		parentSpanContext, isExists2 := c.Get("ParentSpanContext")
		if isExists2 {
			span := opentracing.StartSpan(
				operationName,
				opentracing.ChildOf(parentSpanContext.(opentracing.SpanContext)),
				opentracing.Tags{},
			)
			tenantID := c.Request.Header.Get("tenantId")
			if tenantID != "" {
				span.SetTag("tenantId", tenantID)
			}

			shopCode := c.Request.Header.Get("shopCode")
			if shopCode != "" {
				span.SetTag("shopCode", shopCode)
			}

			// injectErr := tracer.(opentracing.Tracer).Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
			// if injectErr != nil {
			// 	span.Finish()
			// 	return nil, false
			// }
			return span, true
		}
	}
	return nil, false
}

//GetNewSpanFromContextWithParent 获取新的Span用来记录
func GetNewSpanFromContextWithParent(c *gin.Context, operationName string, tracer opentracing.Tracer) (opentracing.Span, bool) {
	if c != nil {
		// tracer, isExists1 := c.Get("Tracer")
		rootSpanContext, isRootExists := c.Get("ParentSpanContext")
		if isRootExists {

			span := tracer.StartSpan(
				operationName,
				opentracing.ChildOf(rootSpanContext.(opentracing.SpanContext)),
				opentracing.Tags{},
			)

			tenantID := c.Request.Header.Get("tenantId")
			if tenantID != "" {
				span.SetTag("tenantId", tenantID)
			}

			shopCode := c.Request.Header.Get("shopCode")
			if shopCode != "" {
				span.SetTag("shopCode", shopCode)
			}

			return span, true
		}
	}
	return nil, false
}

//GetNewMongoSpan 获取新的Span用来记录
func GetDBMongoSpan(c *gin.Context, operationName string, args ...string) (opentracing.Span, bool) {
	if dbMongoTracer.Enable {
		if span, isOk := GetNewSpanFromContextWithParent(c, operationName, dbMongoTracer.Tracer); isOk {
			span.SetTag("db.Type", spanDBMongo)
			if len(args) >= 1 {
				span.SetTag("db.DBName", args[0])
			}
			if len(args) >= 2 {
				span.SetTag("db.CollName", args[1])
			}
			if len(args) >= 3 {
				span.SetTag("db.Cmd", args[2])
			}
			return span, true
		}
	}
	return nil, false
}

//GetDBMSQLSpan 获取新的Span用来记录  参数1 dbname  参数2 cmd
func GetDBMSQLSpan(c *gin.Context, operationName string, args ...string) (opentracing.Span, bool) {
	if dbMssqlTracer.Enable {
		if span, isOk := GetNewSpanFromContextWithParent(c, operationName, dbMssqlTracer.Tracer); isOk {
			span.SetTag("db.Type", spanDBMSSQL)
			if len(args) >= 1 {
				span.SetTag("db.DBName", args[0])
			}
			if len(args) >= 2 {
				span.SetTag("db.Table", args[1])
			}
			if len(args) >= 3 {
				span.SetTag("db.Cmd", args[2])
			}
			return span, true
		}
	}
	return nil, false
}

//GetDBMSQLSpan 获取新的Span用来记录
func GetDBPGSpan(c *gin.Context, operationName string, args ...string) (opentracing.Span, bool) {
	if dbPgTracer.Enable {
		if span, isOk := GetNewSpanFromContextWithParent(c, operationName, dbPgTracer.Tracer); isOk {
			span.SetTag("db.Type", spanDBPG)
			if len(args) >= 1 {
				span.SetTag("db.DBName", args[0])
			}
			if len(args) >= 2 {
				span.SetTag("db.Table", args[1])
			}
			if len(args) >= 3 {
				span.SetTag("db.Cmd", args[2])
			}
			return span, true
		}
	}
	return nil, false
}

//GetDBMSQLSpan 获取新的Span用来记录
func GetDBRedisSpan(c *gin.Context, operationName string, args ...string) (opentracing.Span, bool) {
	if dbRedisTracer.Enable {
		if span, isOk := GetNewSpanFromContextWithParent(c, operationName, dbRedisTracer.Tracer); isOk {
			span.SetTag("db.Type", spanDBREDIS)
			if len(args) >= 1 {
				span.SetTag("db.Key", args[0])
			}
			return span, true
		}
	}
	return nil, false
}

// ForeachKey implements ForeachKey of opentracing.TextMapReader
func (c MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range c.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
func (c MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

// ClientInterceptor grpc client
func ClientInterceptor(tracer opentracing.Tracer, spanContext opentracing.SpanContext) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string,
		req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		span := opentracing.StartSpan(
			"call gRPC",
			opentracing.ChildOf(spanContext),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
			ext.SpanKindRPCClient,
		)

		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		err := tracer.Inject(span.Context(), opentracing.TextMap, MDReaderWriter{md})
		if err != nil {
			span.LogFields(log.String("inject-error", err.Error()))
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		err = invoker(newCtx, method, req, reply, cc, opts...)
		if err != nil {
			span.LogFields(log.String("call-error", err.Error()))
		}
		return err
	}
}
