package convert

import "time"

//LocalTime Date
type LocalTime time.Time

const (
	localtimeFormart = "15:04:05"
)

//UnmarshalJSON 反序列化
func (t *LocalTime) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+localtimeFormart+`"`, string(data), time.Local)
	*t = LocalTime(now)
	return
}

//MarshalJSON 序列化
func (t LocalTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(localtimeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, localtimeFormart)
	b = append(b, '"')
	return b, nil
}

//String 字符串
func (t LocalTime) String() string {
	return time.Time(t).Format(localtimeFormart)
}

//Time 字符串
func (t LocalTime) Time() *time.Time {
	tt := time.Time(t)
	return &tt
}

//LocaltimeToTime 类型转换
func LocaltimeToTime(ti *time.Time) *LocalTime {
	if ti == nil {
		return nil
	} else {
		tt := LocalTime(*ti)
		return &tt
	}

}
