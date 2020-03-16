package convert

import (
	"strconv"
	"strings"
	"time"
)

type LocalDatetime time.Time

const (
	localDatetimeFormart = "2006-01-02T15:04:05"
)

//UnmarshalJSON 反序列化
func (t *LocalDatetime) UnmarshalJSON(data []byte) (err error) {

	// fmt.Println(string(data))
	timeStr := string(data)
	timeStr = strings.Replace(timeStr, "\"", "", -1)
	year, err := strconv.Atoi(timeStr[0:4])
	month, err := strconv.Atoi(timeStr[5:7])
	day, err := strconv.Atoi(timeStr[8:10])
	hour, err := strconv.Atoi(timeStr[11:13])
	min, err := strconv.Atoi(timeStr[14:16])
	sec, err := strconv.Atoi(timeStr[17:19])
	nsec := 0
	if len(timeStr) > 20 {
		nsec, err = strconv.Atoi(timeStr[20:len(timeStr)])
	}

	nsec = nsec * 1000000
	now := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local)
	//now, err := time.ParseInLocation(`"`+timeFormart+`"`, timeStr, time.Local)
	*t = LocalDatetime(now)
	return
}

//MarshalJSON 序列化
func (t LocalDatetime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(localDatetimeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, localDatetimeFormart)
	b = append(b, '"')
	return b, nil
}

//String 字符串
func (t LocalDatetime) String() string {
	return time.Time(t).Format(localDatetimeFormart)
}

//Time 字符串
func (t LocalDatetime) Time() *time.Time {
	tt := time.Time(t)
	return &tt
}

//LocalDatetimeToTime 类型转换
func LocalDatetimeToTime(ti *time.Time) *LocalDatetime {
	if ti == nil {
		return nil
	} else {
		tt := LocalDatetime(*ti)
		return &tt
	}

}
