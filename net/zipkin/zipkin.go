package zipkin

import (
	"github.com/openzipkin/zipkin-go"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

type Config struct {
	ServiceName string //服务名称
	IP          string //本服务IP
	EnpoitUrl   string //zipkin  服务器地址
}

var Tracer *zipkin.Tracer

func NewTracer(config Config) {
	Tracer = GetTracer(config.ServiceName, config.IP, config.EnpoitUrl)
}

func GetTracer(serviceName string, ip string, enpoitUrl string) *zipkin.Tracer {
	// create a reporter to be used by the tracer
	reporter := httpreporter.NewReporter(enpoitUrl)

	// set-up the local endpoint for our service
	endpoint, _ := zipkin.NewEndpoint(serviceName, ip)

	// set-up our sampling strategy
	sampler := zipkin.NewModuloSampler(1)

	// initialize the tracer
	tracer, _ := zipkin.NewTracer(
		reporter,
		zipkin.WithLocalEndpoint(endpoint),
		zipkin.WithSampler(sampler),
	)
	return tracer
}
