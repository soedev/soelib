package utils

import (
	"errors"
	"time"
)

//InquiryDateDTO 返回时间
type InquiryDateDTO struct {
	StartDate string `json:"startDate"` //开始日期
	EndDate   string `json:"endDate"`   //结束日期
	StartTime string `json:"startTime"` //开始营业时间
	EndTime   string `json:"endTime"`   //结束营业时间
	ShopID    string `json:"shopId"`
}

//GetShopBussinessDateTime 获取店内营业时间算法
func GetShopBussinessDateTime(shopStartTime, shopEndTime, stringStartDate, stringEndDate string, isMinusDay bool) (startDateTime, endDateTime string) {
	nowDateTime := time.Now()
	nowDateString := nowDateTime.Format("2006-01-02")
	nowTimeString := nowDateTime.Format("15:04:05")
	var startDate time.Time
	var endDate time.Time
	if isMinusDay {
		if stringStartDate == stringEndDate {
			if stringStartDate == nowDateString {
				endDate, _ = time.Parse("2006-01-02", stringEndDate)
				startDate, _ = time.Parse("2006-01-02", stringStartDate)
				if nowTimeString < shopStartTime {
					decDay, _ := time.ParseDuration("-24h")
					startDate = startDate.Add(decDay)
					endDate = endDate.Add(decDay)
					stringStartDate = startDate.Format("2006-01-02")
					stringEndDate = endDate.Format("2006-01-02")
				}
			}
		}
	}
	var theDate time.Time
	if stringStartDate == stringEndDate {
		theDate, _ = time.Parse("2006-01-02", stringStartDate)
	} else {
		theDate, _ = time.Parse("2006-01-02", stringEndDate)
	}
	startDateTime = stringStartDate + " " + shopStartTime
	if shopStartTime < shopEndTime {
		endDateTime = stringEndDate + " " + shopEndTime
	} else {
		decDay, _ := time.ParseDuration("24h")
		stringEndDate = theDate.Add(decDay).Format("2006-01-02")
		endDateTime = stringEndDate + " " + shopEndTime
	}
	return startDateTime, endDateTime
}

//GetBussinessDateTime 获取营业时间
func GetBussinessDateTime(startTime, endTime string) (string, string) {
	t := time.Now()
	// cstZone, _ := time.ParseDuration("8h")
	// t = t.Add(cstZone)

	startDateTime := ""
	endDateTime := ""
	if startTime <= endTime {
		startDateTime = t.Format("2006-01-02") + " " + startTime
		endDateTime = t.Format("2006-01-02") + " " + endTime
	} else {
		searchTime := t.Format("15:04:05")
		if searchTime > endTime {
			addDay, _ := time.ParseDuration("24h")
			startDateTime = t.Format("2006-01-02") + " " + startTime
			endDateTime = t.Add(addDay).Format("2006-01-02") + " " + endTime
		} else {
			decDay, _ := time.ParseDuration("-24h")
			startDateTime = t.Add(decDay).Format("2006-01-02") + " " + startTime
			endDateTime = t.Format("2006-01-02") + " " + endTime
		}
	}
	return startDateTime, endDateTime
}

//GetBusinessDateDTO 获取营业时间 startDate(开始日期)，endDate(结束日期)，startTime(开始营业时间)，endTime(结束营业时间)
func GetBusinessDateDTO(startDate, endDate, startTime, endTime string, isMinusDay bool) (inquiryDateDTO InquiryDateDTO, err error) {
	startDateTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return inquiryDateDTO, errors.New("开始日期传入有误")
	}
	endDateTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return inquiryDateDTO, errors.New("结束日期传入有误")
	}
	start, err := time.Parse("15:04:05", startTime)
	if err != nil {
		return inquiryDateDTO, errors.New("开始时间传入有误")
	}
	end, err := time.Parse("15:04:05", endTime)
	if err != nil {
		return inquiryDateDTO, errors.New("结束时间传入有误")
	}
	//没到营业时间是否减一天
	if isMinusDay {
		now := time.Now()
		searchTime := now.Format("15:04:05")
		if searchTime > endTime {
			addDay, _ := time.ParseDuration("24h")
			startDate = startDateTime.Format("2006-01-02")
			endDate = endDateTime.Add(addDay).Format("2006-01-02")
		} else {
			decDay, _ := time.ParseDuration("-24h")
			startDate = startDateTime.Add(decDay).Format("2006-01-02")
			endDate = endDateTime.Format("2006-01-02")
		}
	}

	var theDate time.Time
	if startDate == endDate {
		theDate = startDateTime
	} else {
		theDate = endDateTime
	}
	inquiryDateDTO.StartDate = startDate + " " + startTime
	if start.Before(end) {
		inquiryDateDTO.EndDate = endDate + " " + endTime
	} else {
		addDate, _ := time.ParseDuration("24h")
		newEndDate := theDate.Add(addDate)
		inquiryDateDTO.EndDate = newEndDate.Format("2006-01-02") + " " + endTime
	}
	return inquiryDateDTO, nil
}
