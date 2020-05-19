package soetrace

import (
	"github.com/openzipkin/zipkin-go"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

type ZKTracerConfig struct {
	ServiceName string //服务名称
	IP          string //本服务IP
	EnpoitUrl   string //zipkin  服务器地址
}

var Tracer *zipkin.Tracer

func NewZipkinTracer(config ZKTracerConfig) {
	// create a reporter to be used by the tracer
	reporter := httpreporter.NewReporter(config.EnpoitUrl)

	// set-up the local endpoint for our service
	endpoint, _ := zipkin.NewEndpoint(config.ServiceName, config.IP)

	// set-up our sampling strategy
	sampler := zipkin.NewModuloSampler(1)

	// initialize the tracer
	Tracer, _ = zipkin.NewTracer(
		reporter,
		zipkin.WithLocalEndpoint(endpoint),
		zipkin.WithSampler(sampler),
	)
}
