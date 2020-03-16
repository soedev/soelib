package utils

import (
	"fmt"
	"testing"
)

func TestGetShopBussinessDateTime(t *testing.T) {
	startDateTime, endDateTime := GetShopBussinessDateTime("07:00:00", "09:59:59", "2019-09-20", "2019-09-20", false)
	fmt.Println(">>>>>>开始时间：" + startDateTime)
	fmt.Println(">>>>>>>结束时间：" + endDateTime)
}
