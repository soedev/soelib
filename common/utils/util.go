package utils

import (
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/digitalocean/go-smbios/smbios"
	"github.com/ulule/deepcopier"
)

const (
	cOTNoOddment = 0 //不去零
	cOTFrac      = 1 //----去零
	cOTAdd       = 2 //----加零
	cOTRound     = 3 //----四去五入
	cOTTrunc     = 4 //----直接切去小数
)

//CalcOddment 取零方式计算结果
func CalcOddment(oddmentType int, value float64) float64 {
	result := value
	switch oddmentType {
	case cOTFrac:
		result = math.Floor(value)
	case cOTAdd:
		result = math.Ceil(value)
	case cOTRound:
		result = math.Floor(value + 0.5)
	case cOTTrunc:
		result = math.Trunc(value)
	}

	return result

}

//检测对象是否未 nil
func IsNil(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}

//NumberToChinese 将阿拉伯数字转中文数字
func NumberToChinese(str string) (numStr string, err error) {
	chineses := []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	n := strings.Count(str, "") - 1
	var num string
	for i := 0; i < n; i++ {
		numberStr := string([]rune(str)[i : i+1])
		bol, _ := regexp.Match(`\d`, []byte(numberStr))
		if bol == true {
			number, err := strconv.Atoi(numberStr)
			if err != nil {
				return "", errors.New("转中文数字出错")
			}
			chinese := chineses[number]
			num += chinese
		} else {
			num += numberStr
		}
	}
	return string(num), nil
}

//读取主板序列号 GetSystemUUID
func GetSystemUUID() string {
	rc, _, err := smbios.Stream()
	if err != nil {
		log.Printf("获取系统uuid出错:%s", err.Error())
		return ""
	}
	// Be sure to close the stream!
	defer rc.Close()

	// Decode SMBIOS structures from the stream.
	d := smbios.NewDecoder(rc)
	ss, err := d.Decode()
	if err != nil {
		log.Printf("获取系统uuid出错:%s", err.Error())
		return ""
	}
	for _, s := range ss {
		if s.Header.Type == 1 {
			uuid := s.Formatted[4:20]
			return strings.ToUpper(hex.EncodeToString(uuid))
		}
	}
	return ""
}

//TenToTwo　二进制转十进制
func TenToTwo(two string) (ten int) {
	var stnum, conum float64 = 0, 2
	stack := list.New()
	for _, c := range two {
		// 入栈 type rune
		stack.PushBack(c)
	}
	// 出栈
	for e := stack.Back(); e != nil; e = e.Prev() {
		// rune是int32的别名
		v := e.Value.(int32)
		ten += int(v-48) * int(math.Pow(conum, stnum))
		stnum++
	}
	return ten
}

func DecConvertToX(n, num int) (string, error) {
	if n < 0 {
		return strconv.Itoa(n), errors.New("只支持正整数")
	}
	if num != 2 && num != 8 && num != 16 {
		return strconv.Itoa(n), errors.New("只支持二、八、十六进制的转换")
	}
	result := ""
	h := map[int]string{
		0:  "0",
		1:  "1",
		2:  "2",
		3:  "3",
		4:  "4",
		5:  "5",
		6:  "6",
		7:  "7",
		8:  "8",
		9:  "9",
		10: "A",
		11: "B",
		12: "C",
		13: "D",
		14: "E",
		15: "F",
	}
	for ; n > 0; n /= num {
		lsb := h[n%num]
		result = lsb + result
	}
	return result, nil
}

//Decimal 保留两位小数
func Decimal(value float64) float64 {
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}

//ParseInt ParseInt
func ParseInt(value interface{}) int {
	var result int
	switch t := value.(type) {
	case string:
		i, err := strconv.Atoi(t)
		if err != nil {
			result = 0
		} else {
			result = i
		}
	case float32:
		result = int(t)
	case float64:
		result = int(t)
	case int:
		result = t
	case int8:
		result = int(t)
	case int32:
		result = int(t)
	case int64:
		result = int(t)
	}
	return result
}

/*
	从JDBC连接串中取得服务器和数据库名
*/
func GetDBInfo(jdbcURL string) (string, string, int) {
	//jdbcUrl := "jdbc:sqlserver://192.168.1.141:1433;databaseName=MERCURYDB"
	//替换jdbc。。。。为空
	jdbcURL = strings.Replace(jdbcURL, "jdbc:sqlserver://", "", 1)
	//fmt.Println(jdbcURL)
	//192.168.1.141:1433;databaseName=MERCURYDB
	a := strings.Index(jdbcURL, ";")
	serverPort := jdbcURL[:a]
	server := strings.Split(serverPort, ":")[0]
	portStr := strings.Split(serverPort, ":")[1]
	//fmt.Println("port:", portStr)
	//fmt.Println("server:", server)
	dbName := strings.Replace(jdbcURL[a:], ";databaseName=", "", 1)
	//fmt.Println("name:", dbName)
	port, _ := strconv.Atoi(portStr)
	return dbName, server, port
}

//CopyStruct 结构体复制(同一字段需要类型相同)
// user:=&User{
//		Name: "qweqwe",
//		Age:  22,
//	}
//user2:=&User2{}
//CopyStruct(user,user2)
func CopyStruct(src interface{}, dest interface{}) {
	deepcopier.Copy(src).To(dest)
}

//CheckIP 校验ip
func CheckIP(ip string) bool {
	addr := strings.Trim(ip, " ")
	regStr := `^(([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.)(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){2}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	if match, _ := regexp.MatchString(regStr, addr); match {
		return true
	}
	return false
}

//GetPageDTO 手动分页
func GetPageDTO(page, pageSize int, i []interface{}) (count, totalPage, startIndex, endIndex int) {
	count = len(i)
	if count%pageSize == 0 {
		totalPage = count / pageSize
	} else {
		totalPage = count/pageSize + 1
	}
	startIndex = (page - 1) * pageSize
	endIndex = startIndex + pageSize
	if endIndex > count {
		endIndex = count
	}
	return count, totalPage, startIndex, endIndex
}

//CheckMobileNumber 验证手机号码，11位数字，1开通，第二位数必须是3456789这些数字之一
func CheckMobileNumber(phone string) bool {
	regex := "^((13[0-9])|(14[0-9])|(15([0-9]|[0-9]))|(17[0-9])|(18[0-9]))\\d{8}$"
	if utf8.RuneCountInString(phone) != 11 {
		return false
	} else {
		check, _ := regexp.MatchString(regex, phone)
		return check
	}
}

//ListDuplicateRemoval 集合去重
func ListDuplicateRemoval(originals interface{}) (interface{}, error) {
	temp := map[string]struct{}{}
	switch slice := originals.(type) {
	case []string:
		result := make([]string, 0, len(originals.([]string)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	case []int64:
		result := make([]int64, 0, len(originals.([]int64)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	case []float64:
		result := make([]float64, 0, len(originals.([]float64)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	default:
		return nil, errors.New("未知类型")
	}
}
