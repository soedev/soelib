package slowInterface

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	alirabbitmq "github.com/soedev/soelib/net/alirabbit"
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
	Type      int                    `json:"type"`
	Datetime  time.Time              `json:"datetime"`
	IsSkip    bool                   `json:"isSkip"`
	Key       string                 `json:"key"`
}

//SlowInterface 慢接口统计
func SlowInterface( tag string, slowTime int,rabbitCon alirabbitmq.Connection) gin.HandlerFunc {
	return GetSlowInterface( tag, slowTime,rabbitCon)
}

func GetSlowInterface( tag string, slowTime int,rabbitCon alirabbitmq.Connection) gin.HandlerFunc {
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

			//times := slowTime * 1000000
			if param.Latency.Seconds()*1 < (time.Second*time.Duration(slowTime)).Seconds() {
				return
			}

			param.Method = c.Request.Method

			param.Path = path

			bytes, _ := json.Marshal(param)
			rabbitCon.SendMessage(bytes,"soesoft.slow.queue")
		}
	}
}
