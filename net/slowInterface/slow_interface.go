package slowInterface

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/soedev/soelib/net/kafka"
	"net/http"
	"strings"
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
func SlowInterface(kafkaServer, tag string, slowTime int) gin.HandlerFunc {
	return GetSlowInterface(kafkaServer, tag, slowTime)
}

func GetSlowInterface(kafkaServer, tag string, slowTime int) gin.HandlerFunc {
	// 选择要跳过的接口
	//notlogged := make([]string, 0)
	//
	//var skip map[string]struct{}
	//
	//if length := len(notlogged); length > 0 {
	//	skip = make(map[string]struct{}, length)
	//
	//	for _, path := range notlogged {
	//		skip[path] = struct{}{}
	//	}
	//}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Log only when path is not being skipped
		//if _, ok := skip[path]; !ok {
		tenantID := c.Request.Header.Get("tenantId")
		shopCode := c.Request.Header.Get("shopCode")
		if tenantID == "" || shopCode == "" {
			return
		}

		param := SlowInterfaceParams{
			Key:      strings.ReplaceAll(uuid.New().String(), "-", ""),
			Request:  c.Request,
			Keys:     c.Keys,
			Tag:      tag,
			TenantID: tenantID,
			ShopCode: shopCode,
			Datetime: start,
			Path:     path,
			Method:   c.Request.Method,
			Type:     1,
		}

		bytes, _ := json.Marshal(param)
		// 发送到kafka，记录访问时间
		kafka.SendSarama(kafkaServer, "slow_interface", bytes)

		// Process request
		c.Next()

		// 计算访问时间
		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)

		times := slowTime * 1000000
		if param.Latency < time.Duration(times) {
			param.IsSkip = true
		}
		param.Type = 0
		param.Datetime = param.TimeStamp

		bytes, _ = json.Marshal(param)
		// 发送到kafka，记录返回时间
		kafka.SendSarama(kafkaServer, "slow_interface", bytes)
		//}
	}
}
