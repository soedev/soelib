package app

import (
	"github.com/getsentry/raven-go"
	"github.com/gin-gonic/gin"
	"github.com/soedev/soelib/net/e"
	"net/http"
)

//Gin gin
type Gin struct {
	C           *gin.Context
	ServiceName string
	TenantID    string
	ShopCode    string
	Version     string
	Error       error
	IsServer    bool
}

//Response 统一返回
func (g *Gin) Response(httpCode, errCode int, data interface{}) {
	g.C.JSON(httpCode, gin.H{
		"code": httpCode,
		"msg":  e.GetMsg(errCode),
		"data": data,
	})

	if httpCode != http.StatusOK && httpCode != http.StatusUnauthorized {
		errorKey := g.ServiceName
		if g.ShopCode != "" {
			errorKey = " 门店[" + g.ShopCode + "] " + errorKey
		}
		if g.TenantID != "" {
			errorKey = " 租户[" + g.TenantID + "] " + errorKey
		}
		//	soelog.Logger.Error(errorKey, zap.String("错误", fmt.Sprintf("%v", data)))

		if g.Error != nil {
			raven.CaptureError(g.Error, map[string]string{"ServiceName": g.ServiceName,
				"TenantID": g.TenantID, "ShopCode": g.ShopCode, "Version": g.Version})
		}
	}

	return
}
