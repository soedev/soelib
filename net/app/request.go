package app

import (
	"github.com/astaxie/beego/validation"
	"go.uber.org/zap"
	"log"
)

//MarkErrors 标记错误
func MarkErrors(errors []*validation.Error) {
	for _, err := range errors {
		log.Println(err.Key, zap.String("msg", err.Message))
	}
	return
}
