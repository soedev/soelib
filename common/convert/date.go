package convert

import "time"

//Date Date
type Date time.Time

const (
	timeFormart = "2006-01-02"
)

//UnmarshalJSON 反序列化
func (t *Date) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+timeFormart+`"`, string(data), time.Local)
	*t = Date(now)
	return
}

//MarshalJSON 序列化
func (t Date) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(timeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, timeFormart)
	b = append(b, '"')
	return b, nil
}

//String 字符串
func (t Date) String() string {
	return time.Time(t).Format(timeFormart)
}

//TimeToDate time 转 date
func TimeToDate(ti *time.Time) *Date {
	if ti == nil {
		return nil
	} else {
		tt := Date(*ti)
		return &tt
	}

}

//Time 字符串
func (t Date) Time() *time.Time {
	tt := time.Time(t)
	return &tt
}
