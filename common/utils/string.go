package utils

/**
  string   字符串逻辑处理类
*/

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Lofanmi/pinyin-golang/pinyin"
	"golang.org/x/text/encoding/simplifiedchinese"
)

//GBKEncoder 字符串转成 GBK 格式
func GBKEncoder(value string) (string, bool) {
	str, err := simplifiedchinese.GBK.NewEncoder().String(value)
	if err != nil {
		return "", false
	}
	return str, true
}

//GBKEncoderUseLen 字符串转成 GBK格式 并截取固定长度
func GBKEncoderUseLen(value string, alen int) (string, bool) {
	content := []rune(value)
	if len(content) > alen {
		return GBKEncoder(string(content[:alen]))
	} else {
		return GBKEncoder(value)
	}
}

//StrRightIsLenThan 如果超出指定大小将从右边截取
func StrRightIsLenThan(value string, alen int) string {
	content := []rune(value)
	cLen := len(content)
	if cLen > alen {
		return string(content[cLen-alen : cLen])
	} else {
		return value
	}
}

//StrRightIsLenThan 如果超出指定大小将从左边截取
func StrLeftIsLenThan(value string, alen int) string {
	content := []rune(value)
	if len(content) > alen {
		return string(content[0:alen])
	} else {
		return value
	}
}

//CheckHaveSubString 检查被分割符分割的字符串中是否包含制定的字符串
func CheckHaveSubString(aSourceStr, aSubStr, aDelimiter string) bool {
	if aSourceStr == "" {
		return false
	}
	for _, k := range strings.Split(aSourceStr, aDelimiter) {
		if k == aSubStr {
			return true
		}
	}
	return false
}

func StringsContains(array []string, val string) (exist bool) {
	exist = false
	for i := 0; i < len(array); i++ {
		if array[i] == val {
			exist = true
			return exist
		}
	}
	return exist
}

func WipeOutEle(strs []string, val string) (arrayStr []string) {
	for _, ele := range strs {
		if ele != val {
			arrayStr = append(arrayStr, ele)
		}
	}
	return arrayStr
}

//ChineseToAbc 将中文字符串转拼音首字母
func ChineseToAbc(chinese string) string {
	str := pinyin.NewDict()
	en := str.Abbr(chinese, "")
	return en
}

//SubstrByByte 按字节截取字符串
func SubstrByByte(str string, length int) string {
	bs := []byte(str)[:length]
	bl := 0
	for i := len(bs) - 1; i >= 0; i-- {
		switch {
		case bs[i] >= 0 && bs[i] <= 127:
			return string(bs[:i+1])
		case bs[i] >= 128 && bs[i] <= 191:
			bl++
		case bs[i] >= 192 && bs[i] <= 253:
			cl := 0
			switch {
			case bs[i]&252 == 252:
				cl = 6
			case bs[i]&248 == 248:
				cl = 5
			case bs[i]&240 == 240:
				cl = 4
			case bs[i]&224 == 224:
				cl = 3
			default:
				cl = 2
			}
			if bl+1 == cl {
				return string(bs[:i+cl])
			}
			return string(bs[:i])
		}
	}
	return ""
}

//string数组转string  例：[]string{"1", "2", "3", "4"} ==> '1','2','3','4'
func StringList2Str(list []string) string {
	str := ""
	if list != nil && len(list) > 0 {
		for i := 0; i < len(list); i++ {
			if i == len(list)-1 {
				str = str + list[i] + "'"
			} else {
				str = str + list[i] + "','"
			}
		}
		str = "'" + str
	}
	return str
}

// FormatterStr 将'0,1,2'转换成'0','1','2'
func FormatterStr(value string) string {
	strArr := strings.Split(value, ",")
	key := ""
	for index, temp := range strArr {
		if temp != "" {
			temp = "'" + temp + "'"
			key += temp
			if index+1 != len(strArr) {
				key += ","
			}
		}
	}
	return key
}

// ToNullString 格式化
func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ToNullFloat64 格式化
func ToNullFloat64(s float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: s, Valid: true}
}

// ToNullInt64 格式化
func ToNullInt64(s string) sql.NullInt64 {
	i, err := strconv.Atoi(s)
	return sql.NullInt64{Int64: int64(i), Valid: err == nil}
}

// 基础的字符串类型转换
func String(i interface{}) string {
	if i == nil {
		return ""
	}
	switch value := i.(type) {
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.Itoa(int(value))
	case int16:
		return strconv.Itoa(int(value))
	case int32:
		return strconv.Itoa(int(value))
	case int64:
		return strconv.Itoa(int(value))
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(uint64(value), 10)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	case []byte:
		return string(value)
	default:
		// 默认使用json进行字符串转换
		jsonContent, _ := json.Marshal(value)
		return string(jsonContent)
	}
}
