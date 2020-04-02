package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestGetShopBussinessDateTime(t *testing.T) {
	startDateTime, endDateTime := GetShopBussinessDateTime("07:00:00", "09:59:59", "2019-09-20", "2019-09-20", false)
	fmt.Println(">>>>>>开始时间：" + startDateTime)
	fmt.Println(">>>>>>>结束时间：" + endDateTime)
}

func TestIsNearStartTime(t *testing.T) {
	ti,_ := time.Parse("15:04:05","20:00:00")
	s,_ := time.Parse("15:04:05","22:00:00")
	e,_ := time.Parse("15:04:05","10:59:59")
	b:=IsNearStartTime(&ti,&s,&e)
	fmt.Println(b)
}