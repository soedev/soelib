package convert

import (
	"strconv"
	"strings"
	"time"
)

//Datetime Datetime
type Datetime time.Time

const (
	datetimeFormart = "2006-01-02 15:04:05.999999999"
)

//UnmarshalJSON 反序列化
func (t *Datetime) UnmarshalJSON(data []byte) (err error) {

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
	*t = Datetime(now)
	return
}

//MarshalJSON 序列化
func (t Datetime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(datetimeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, datetimeFormart)
	b = append(b, '"')
	return b, nil
}

//String 字符串
func (t Datetime) String() string {
	return time.Time(t).Format(datetimeFormart)
}

//Time 字符串
func (t Datetime) Time() *time.Time {
	tt := time.Time(t)
	return &tt
}

//DatetimeToTime 类型转换
func DatetimeToTime(ti *time.Time) *Datetime {
	if ti == nil {
		return nil
	} else {
		tt := Datetime(*ti)
		return &tt
	}

}
