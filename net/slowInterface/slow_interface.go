package slowInterface

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/soedev/soelib/net/kafka"
	"net/http"
	"time"
)

type SlowInterfaceParams struct {
	Request   *http.Request          `json:"-"`
	Keys      map[string]interface{} `json:"-"`
	TimeStamp time.Time              `json:"-"`
	Latency   time.Duration          `json:"latency"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	Tag       string                 `json:"tag"`
	TenantID  string                 `json:"tenantId"`
	ShopCode  string                 `json:"shopCode"`
}

//SlowInterface 慢接口统计
func SlowInterface(kafkaServer, tag string, slowTime int) gin.HandlerFunc {
	return GetSlowInterface(kafkaServer, tag, slowTime)
}

func GetSlowInterface(kafkaServer, tag string, slowTime int) gin.HandlerFunc {
	notlogged := make([]string, 0)

	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			tenantID := c.Request.Header.Get("tenantId")
			shopCode := c.Request.Header.Get("shopCode")
			if tenantID == "" || shopCode == "" {
				return
			}

			param := SlowInterfaceParams{
				Request:  c.Request,
				Keys:     c.Keys,
				Tag:      tag,
				TenantID: tenantID,
				ShopCode: shopCode,
			}

			// Stop timer
			param.TimeStamp = time.Now()
			param.Latency = param.TimeStamp.Sub(start)

			times := slowTime * 1000000
			if param.Latency < time.Duration(times) {
				return
			}

			param.Method = c.Request.Method

			param.Path = path

			bytes, _ := json.Marshal(param)
			// 发送到kafka
			kafka.SendSarama(kafkaServer, "slow_interface", bytes)
		}
	}
}
