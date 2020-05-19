package soetrace

import (
	"github.com/openzipkin/zipkin-go"
	"testing"
	"time"
)

//测试手工埋点案例
func TestNewTracer(t *testing.T) {
	config := ZKTracerConfig{
		ServiceName: "demoService",
		IP:          "192.168.1.129:80",
		EnpoitUrl:   "http://192.168.1.206:32019/api/v2/spans",
	}
	NewZipkinTracer(config)
	// tracer can now be used to create spans.
	span := Tracer.StartSpan("会员查询")
	// ... do some work ...
	doOne(10 * time.Second) //模拟需要花费的时间
	span.Finish()

	childSpan := Tracer.StartSpan("查询积分", zipkin.Parent(span.Context()))
	// ... do some work ...
	doOne(3 * time.Second) //模拟需要花费的时间
	childSpan.Finish()

	childSpan1 := Tracer.StartSpan("查询消费", zipkin.Parent(span.Context()))
	// ... do some work ...
	doOne(5 * time.Second) //模拟需要花费的时间
	childSpan1.Finish()

	span.Finish()
	// Output:

}

func doOne(duration time.Duration) {
	time.Sleep(duration)
}
