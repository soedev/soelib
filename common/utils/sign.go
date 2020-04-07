package utils

/**
  Sign    MD5签名 工具类
*/
import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

//ValidSign  签名效验正确性
func ValidSign(sign, srcdata interface{}, bizkey string) bool {
	curSign := Getsign(srcdata, bizkey)
	if curSign == "" || sign == "" {
		return false
	}
	return curSign == sign
}

// Getsign 生成验签
func Getsign(srcdata interface{}, bizkey string) string {
	md5ctx := md5.New()
	switch v := reflect.ValueOf(srcdata); v.Kind() {
	case reflect.String:
		md5ctx.Write([]byte(v.String() + bizkey))
		return hex.EncodeToString(md5ctx.Sum(nil))
	case reflect.Map:
		orderStr := orderParam(v.Interface(), bizkey)
		md5ctx.Write([]byte(orderStr))
		return hex.EncodeToString(md5ctx.Sum(nil))
	case reflect.Struct:
		orderStr := Struct2map(v.Interface(), bizkey)
		md5ctx.Write([]byte(orderStr))
		return hex.EncodeToString(md5ctx.Sum(nil))
	default:
		return ""
	}
}

//orderParam 排序参数 拼装成字符串
func orderParam(source interface{}, bizKey string) (returnStr string) {
	switch v := source.(type) {
	case map[string]string:
		keys := make([]string, 0, len(v))

		for k := range v {
			if strings.ToLower(k) == "sign" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i int, j int) bool {
			return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
		})
		var buf bytes.Buffer
		for _, k := range keys {
			if v[k] == "" {
				continue
			}
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}

			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v[k])
		}
		buf.WriteString(bizKey)
		returnStr = buf.String()
	case map[string]interface{}:
		keys := make([]string, 0, len(v))

		for k := range v {
			if strings.ToLower(k) == "sign" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i int, j int) bool {
			return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
		})
		var buf bytes.Buffer
		for _, k := range keys {
			if v[k] == "" {
				continue
			}
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}

			buf.WriteString(k)
			buf.WriteByte('=')
			// buf.WriteString(srcmap[k])
			switch vv := v[k].(type) {
			case string:
				buf.WriteString(vv)
			case int:
				buf.WriteString(strconv.FormatInt(int64(vv), 10))
			default:
				panic("params type not supported")
			}
		}
		buf.WriteString(bizKey)
		returnStr = buf.String()
	}
	// fmt.Println(returnStr)
	return
}

//Struct2map Struct转换成Map
func Struct2map(content interface{}, bizKey string) string {
	var tempArr []string
	temString := ""
	var val map[string]string
	if marshalContent, err := json.Marshal(content); err != nil {
		fmt.Println(err)
	} else {
		d := json.NewDecoder(bytes.NewBuffer(marshalContent))
		d.UseNumber()
		if err := d.Decode(&val); err != nil {
			//fmt.Println(err)
		} else {
			for k, v := range val {
				val[k] = v
			}
		}
	}
	i := 0
	for k, v := range val {
		// 去除冗余未赋值struct
		if v == "" {
			continue
		}
		i++
		tempArr = append(tempArr, k+"="+v)
	}
	sort.Slice(tempArr, func(i int, j int) bool {
		return strings.ToLower(tempArr[i]) < strings.ToLower(tempArr[j])
	})
	for n, v := range tempArr {
		if n+1 < len(tempArr) {
			temString = temString + v + "&"
		} else {
			temString = temString + v + "&key=" + bizKey
		}
	}
	return temString
}

func GetXunLianTemp(content interface{}, key string) string {
	var tempArr []string
	temString := ""
	var val map[string]string
	if marshalContent, err := json.Marshal(content); err != nil {
		fmt.Println(err)
	} else {
		d := json.NewDecoder(bytes.NewBuffer(marshalContent))
		d.UseNumber()
		if err := d.Decode(&val); err != nil {
			fmt.Println(err)
		} else {
			for k, v := range val {
				val[k] = v
			}
		}
	}
	i := 0
	for k, v := range val {
		// 去除冗余未赋值struct
		if v == "" {
			continue
		}
		i++
		tempArr = append(tempArr, k+"="+v)
	}
	sort.Slice(tempArr, func(i int, j int) bool {
		return strings.ToLower(tempArr[i]) < strings.ToLower(tempArr[j])
	})
	for n, v := range tempArr {
		if n+1 < len(tempArr) {
			temString = temString + v + "&"
		} else {
			temString = temString + v + key
		}
	}
	return temString
}
