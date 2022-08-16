package utils

/**
  date  时间日期处理工具类
*/

import (
	"bufio"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DateStyle string

const (
	MM_DD                   = "MM-dd"
	YYYYMM                  = "yyyyMM"
	YYYY_MM                 = "yyyy-MM"
	YYYY_MM_DD              = "yyyy-MM-dd"
	YYYYMMDD                = "yyyyMMdd"
	YYYYMMDDHHMMSS          = "yyyyMMddHHmmss"
	YYYYMMDDHHMM            = "yyyyMMddHHmm"
	YYYYMMDDHH              = "yyyyMMddHH"
	YYMMDDHHMM              = "yyMMddHHmm"
	MM_DD_HH_MM             = "MM-dd HH:mm"
	MM_DD_HH_MM_SS          = "MM-dd HH:mm:ss"
	YYYY_MM_DD_HH_MM        = "yyyy-MM-dd HH:mm"
	YYYY_MM_DD_HH_MM_SS     = "yyyy-MM-dd HH:mm:ss"
	YYYY_MM_DD_HH_MM_SS_SSS = "yyyy-MM-dd HH:mm:ss.SSS"

	MM_DD_EN                   = "MM/dd"
	YYYY_MM_EN                 = "yyyy/MM"
	YYYY_MM_DD_EN              = "yyyy/MM/dd"
	MM_DD_HH_MM_EN             = "MM/dd HH:mm"
	MM_DD_HH_MM_SS_EN          = "MM/dd HH:mm:ss"
	YYYY_MM_DD_HH_MM_EN        = "yyyy/MM/dd HH:mm"
	YYYY_MM_DD_HH_MM_SS_EN     = "yyyy/MM/dd HH:mm:ss"
	YYYY_MM_DD_HH_MM_SS_SSS_EN = "yyyy/MM/dd HH:mm:ss.SSS"

	MM_DD_CN               = "MM月dd日"
	YYYY_MM_CN             = "yyyy年MM月"
	YYYY_MM_DD_CN          = "yyyy年MM月dd日"
	MM_DD_HH_MM_CN         = "MM月dd日 HH:mm"
	MM_DD_HH_MM_SS_CN      = "MM月dd日 HH:mm:ss"
	YYYY_MM_DD_HH_MM_CN    = "yyyy年MM月dd日 HH:mm"
	YYYY_MM_DD_HH_MM_SS_CN = "yyyy年MM月dd日 HH:mm:ss"

	HH_MM    = "HH:mm"
	HH_MM_SS = "HH:mm:ss"
)

//DateBetweenMinutes 字符串时间与当前时间相差分钟数
func DateBetweenMinutes(value string) (float64, bool) {
	dateTime, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err == nil {
		btTime := time.Now().Sub(dateTime).Minutes()
		return btTime, true
	}
	return 0.00, false
}

//FormatDate 日期转字符串
func FormatDate(date time.Time, dateStyle DateStyle) string {
	layout := string(dateStyle)
	layout = strings.Replace(layout, "yyyy", "2006", 1)
	layout = strings.Replace(layout, "yy", "06", 1)
	layout = strings.Replace(layout, "MM", "01", 1)
	layout = strings.Replace(layout, "dd", "02", 1)
	layout = strings.Replace(layout, "HH", "15", 1)
	layout = strings.Replace(layout, "mm", "04", 1)
	layout = strings.Replace(layout, "ss", "05", 1)
	layout = strings.Replace(layout, "SSS", "000", -1)
	return date.Format(layout)
}

//日期格式日期加 天数
func DateTimeIncDay(dateTime *time.Time, day int) (oDate string) {
	dayStr := strconv.Itoa(day*24) + "h"
	d, _ := time.ParseDuration(dayStr)
	oDateTime := dateTime.Add(d).Format("2006-01-02")
	return oDateTime
}

//DateStringIncDay 字符日期加 天数
func DateStringIncDay(dateTime string, day int) (oDate string) {
	if len(dateTime) <= 10 {
		dateTime = dateTime + " 00:00:00"
	}
	curDateTime, _ := StringDateToDateTime(dateTime)
	dayStr := strconv.Itoa(day*24) + "h"
	d, _ := time.ParseDuration(dayStr)
	oDateTime := curDateTime.Add(d).Format("2006-01-02")
	return oDateTime
}

//StringDateTimeToDateTime 字符串转为时间格式
func StringDateToDateTime(dateTime string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", dateTime)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

//IsMatchTime 判断时间是否在另一时间段内
func IsMatchTime(t *time.Time, startTime *time.Time, endTime *time.Time) bool {

	isMatch := false

	if t == nil {
		return isMatch
	}

	searchTime := t.Format("15:04:05")

	startPeriodtTime := ""
	if startTime != nil {
		startPeriodtTime = startTime.Format("15:04:05")
	}
	endPeriodTime := ""
	if endTime != nil {
		endPeriodTime = endTime.Format("15:04:05")
	}
	if startPeriodtTime == "" {
		if endPeriodTime == "" {
			isMatch = true
		} else {
			if searchTime <= endPeriodTime {
				isMatch = true
			}
		}
	} else {
		if endPeriodTime == "" {
			if searchTime >= startPeriodtTime {
				isMatch = true
			}
		} else {
			if startPeriodtTime <= endPeriodTime {
				if searchTime >= startPeriodtTime && searchTime <= endPeriodTime {
					isMatch = true
				}
			} else {
				if (searchTime >= startPeriodtTime && searchTime <= "23:59:59") || (searchTime >= "00:00:00" && searchTime <= endPeriodTime) {
					isMatch = true
				}
			}
		}
	}

	return isMatch
}

//StringToDateTime 字符串转时间格式
func StringToDateTime(Type, datetime string) (*time.Time, error) {
	if Type == "datetime" {
		t, err := time.Parse("2006-01-02 15:04:05", datetime)
		if err != nil {
			return nil, errors.New("时间格式传入有误")
		}
		return &t, nil
	} else if Type == "date" {
		t, err := time.Parse("2006-01-02", datetime)
		if err != nil {
			return nil, errors.New("时间格式传入有误")
		}
		return &t, nil
	} else if Type == "time" {
		t, err := time.Parse("15:04:05", datetime)
		if err != nil {
			return nil, errors.New("时间格式传入有误")
		}
		return &t, nil
	}
	return nil, errors.New("未传入时间类型")
}

//DatetimeToString 时间转字符串
func DatetimeToString(Type string, date *time.Time) (dateTime string) {
	if date != nil {
		if Type == "datetime" {
			t := date.Format("2006-01-02 15:04:05")
			return t
		} else if Type == "date" {
			t := date.Format("2006-01-02")
			return t
		} else if Type == "time" {
			t := date.Format("15:04:05")
			return t
		}
	}
	return ""
}

//GetSecondDiffer 获取相差时间
func GetSecondDiffer(startTime, endTime string) int64 {
	var second int64
	t1, err := time.Parse("15:04:05", startTime)
	t2, err := time.Parse("15:04:05", endTime)
	if err == nil {
		diff := t2.Unix() - t1.Unix()

		second = diff
		if second < 0 {
			second = -second
		}
		return second
	}
	return second
}

//IsNearStartTime 检查时间是否接近开始时间
func IsNearStartTime(t *time.Time, startTime *time.Time, endTime *time.Time) bool {
	isNear := false

	//当前时间在凌晨的到上班时间 计算:下班时间到0点的秒数+0点到当前时间的秒数
	if "00:00:00" <= t.Format("15:04:05") && t.Format("15:04:05") < startTime.Format("15:04:05") {
		//获取当前时间离开始时间秒数
		startTimeSecond := GetSecondDiffer(time.Now().Format("15:04:05"), startTime.Format("15:04:05"))

		var endTimeSecond int64
		var zeroSecond int64
		//跨天的上下班时间
		if startTime.Format("15:04:05") > endTime.Format("15:04:05") {
			endTimeSecond = GetSecondDiffer(time.Now().Format("15:04:05"), endTime.Format("15:04:05"))
			zeroSecond = 0
		} else {
			//当前时间到0点时间秒数
			endTimeSecond = GetSecondDiffer(time.Now().Format("15:04:05"), "00:00:00")
			//结束时间到23：59：59点秒数
			zeroSecond = GetSecondDiffer("23:59:59", endTime.Format("15:04:05"))
		}
		if startTimeSecond < endTimeSecond+zeroSecond {
			isNear = true
		}
	} else {
		startTimeSecond := GetSecondDiffer(time.Now().Format("15:04:05"), startTime.Format("15:04:05"))
		endTimeSecond := GetSecondDiffer(time.Now().Format("15:04:05"), endTime.Format("15:04:05"))
		if startTimeSecond < endTimeSecond {
			isNear = true
		}
	}
	return isNear

}

//TimeSubDays 计算两个日期间相差多少天
func TimeSubDays(t1, t2 time.Time) int {

	if t1.Location().String() != t2.Location().String() {
		return -1
	}
	hours := t1.Sub(t2).Hours()

	if hours <= 0 {
		return -1
	}
	// sub hours less than 24
	if hours < 24 {
		// may same day
		t1y, t1m, t1d := t1.Date()
		t2y, t2m, t2d := t2.Date()
		isSameDay := (t1y == t2y && t1m == t2m && t1d == t2d)

		if isSameDay {

			return 0
		} else {
			return 1
		}

	} else {

		if (hours/24)-float64(int(hours/24)) == 0 {
			return int(hours / 24)
		} else {
			return int(hours/24) + 1
		}
	}

}

func TimeBetweenMinutes(startDateTime, endDateTime *time.Time) (float64, error) {
	t := endDateTime.Sub(*startDateTime).Minutes()
	return t, nil
}

//DateTimeBetweenMinutes 时间的相距分钟
func DateTimeBetweenMinutes(startDateTime, endDateTime string) (float64, error) {

	startTime, err := time.Parse("2006-01-02 15:04:05", startDateTime)
	if err != nil {
		return 0, errors.New("开始时间传入有误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", endDateTime)
	if err != nil {
		return 0, errors.New("结束时间传入有误")
	}
	t := endTime.Sub(startTime).Minutes()
	return t, nil
}

//DateTimeBetweenMinutes 时间的相距分钟
func DateTimeBetweenSeconds(startDateTime, endDateTime string) (float64, error) {

	startTime, err := time.Parse("2006-01-02 15:04:05", startDateTime)
	if err != nil {
		return 0, errors.New("开始时间传入有误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", endDateTime)
	if err != nil {
		return 0, errors.New("结束时间传入有误")
	}
	t := endTime.Sub(startTime).Seconds()
	return t, nil
}

//StringDateTimeToDateTime 字符串转为时间格式
func StringDateTimeToDateTime(dateTime string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", dateTime)
	if err != nil {
		return nil, errors.New("时间格式传入有误")
	}
	return &t, nil
}

//DateTimeToTStringDateTime 时间格式转为字符串
func DateTimeToTStringDateTime(date *time.Time) (dateTime string) {
	t := date.Format("2006-01-02T15:04:05")
	return t
}

//DateTimeToTStringDateTime 时间格式转为字符串
func DateTimeToStringDateTime(date *time.Time) (dateTime string) {
	t := date.Format("2006-01-02 15:04:05")
	return t
}

//DateToStringDate 时间格式转为字符串
func DateToStringDate(date *time.Time) (dateTime string) {
	t := date.Format("2006-01-02")
	return t
}

// ConvertToDatetime 格式化
func ConvertToDatetime(s string) *time.Time {
	var datetime *time.Time
	if s != "" {
		datetimeTmp, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
		if err == nil {
			datetime = &datetimeTmp
		}
	}

	return datetime
}

// ConvertTotime 根据时间格式转化
func ConvertTotime(f, t string) *time.Time {
	var datetime *time.Time
	if t != "" && f != "" {
		datetimeTmp, err := time.ParseInLocation(f, t, time.Local)
		if err == nil {
			datetime = &datetimeTmp
		}
	}

	return datetime
}

//CompareTime 比较2个时间
func CompareTime(t1 *time.Time, t2 *time.Time) int {

	if t1 == nil {
		return -1
	}

	if t2 == nil {
		return 1
	}

	if t1.Equal(*t2) {
		return 0
	} else if t1.After(*t2) {
		return 1
	} else {
		return -1
	}
}

//获取指定年月的天数
func GetDayCount(year string, month string) (days int) {
	yearNo := input(year, "^[0-9]{1}[0-9]{3}$")
	monthNo := input(month, "^(0{1}[0-9]{1}|1{1}[0-2]{1})$")
	if monthNo != 2 {
		if monthNo == 4 || monthNo == 6 || monthNo == 9 || monthNo == 11 {
			days = 30
		} else {
			days = 31
		}
	} else {
		if ((yearNo%4) == 0 && (yearNo%100) != 0) || (yearNo%400) == 0 {
			days = 29
		} else {
			days = 28
		}
	}
	return
}

func input(name string, regexpText string) (number int) {
	var validNumber = false
	for !validNumber {
		reader := bufio.NewReader(os.Stdin)
		inputBytes, _, err := reader.ReadLine()
		if err != nil {
			continue
		}
		inputText := string(inputBytes)
		validNumber, err = regexp.MatchString(regexpText, inputText)
		if err != nil {
			continue
		}
		if validNumber {
			number, err = strconv.Atoi(inputText)
			if err != nil {
				continue
			}
		}
	}
	return
}

//日期 加减 指定时间   incTime格式: + - 2h(小时) + - 2m(分钟)
//incTime 合法的单位有"ns"纳秒,"us","µs"、"ms"毫秒、"s"秒、"m"分钟、"h"小时
func DateTimeStringIncTime(dateTime string, incTime string) (oDateTime string) {
	curDateTime := ConvertToDatetime(dateTime)
	t, _ := time.ParseDuration(incTime)
	oDateTime = curDateTime.Add(t).Format("2006-01-02 15:04:05")
	return oDateTime
}

//日期 加减 指定时间   incTime格式: + - 2h(小时) + - 2m(分钟)
//incTime 合法的单位有"ns"纳秒,"us","µs"、"ms"毫秒、"s"秒、"m"分钟、"h"小时
func DateTimeIncTime(dateTime *time.Time, incTime string) (oDateTime string) {
	t, _ := time.ParseDuration(incTime)
	oDateTime = dateTime.Add(t).Format("2006-01-02 15:04:05")
	return oDateTime
}

//通过字符日期获得 时间部分
func FormatTimeByDateString(dateTime string) string {
	time := dateTime[len(dateTime)-8:]
	return time
}

//通过字符日期获得 时间部分
func FormatTimeByDateTime(dateTime *time.Time) string {
	dateTimeString := DateTimeToStringDateTime(dateTime)
	time := dateTimeString[len(dateTimeString)-8:]
	return time
}

//根据不同时间（年，月，日） 计算新时间
func TimeCalculation(old *time.Time, goodTime int, goodUnit string) *time.Time {
	var new time.Time
	if old == nil {
		return &new
	}
	if goodUnit == "YEAR" {
		new = old.AddDate(goodTime, 0, 0)
	} else if goodUnit == "MONTH" {
		new = old.AddDate(0, goodTime, 0)
	} else if goodUnit == "DAY" {
		new = old.AddDate(0, 0, goodTime)
	}
	return &new
}

//TimeBetweenDays 计算天数差
func TimeBetweenDays(t1, t2 *time.Time) (day float64) {
	if t1 == nil || t2 == nil {
		return -1
	}
	a, _ := time.Parse("2006-01-02", DateToStringDate(t1))
	b, _ := time.Parse("2006-01-02", DateToStringDate(t2))
	day = (a.Sub(b)).Hours() / 24
	return day
}
