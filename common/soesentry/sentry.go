package soesentry

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/soedev/soelib/common/soelog"
	"runtime/debug"
	"time"
)
type Sentry struct {
	Open bool
	Dns  string
	Time int64
}
//InitSentry 初始化sentry日志
func InitSentry (config Sentry){
	if config.Open{
		err := sentry.Init(sentry.ClientOptions{
			Dsn: config.Dns,
		})
		if err != nil {
			soelog.Logger.Fatal(fmt.Sprintf("初始化Sentry服务失败:%s", err.Error()))
			return
		}
		sentry.Flush(time.Second*time.Duration(config.Time))
	}
}
//SendSentryLog 记录sentry日志
func SendSentryLog(c *gin.Context,message string){
	tenantID := c.Request.Header.Get("tenantId")

	shopCode := c.Request.Header.Get("shopCode")
	stackMsg := string(debug.Stack())
	clientIP := c.ClientIP()
	url := c.Request.URL.Path

	s := fmt.Sprintf("[Recovery] 时间:%s \n客户:%s \n分店:%s \nIP:%s \nAPI: %s \nrecovered:%s \n%s",
		time.Now().Format("2006-01-02 15:04:05"), tenantID, shopCode, clientIP, url, message, stackMsg)

	soelog.Logger.Error(s)
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.Scope().SetTag("tenantID", tenantID)
		hub.Scope().SetTag("shopCode", shopCode)
		// hub.Scope().SetTag("API", url)
		//hub.CaptureMessage(fmt.Sprintf("%s", err))
		hub.Recover(message)
	}
}