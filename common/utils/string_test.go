package utils

import (
	"fmt"
	"strings"
	"testing"
)

func Test_ChineseToAbc(t *testing.T) {
	abc := ChineseToAbc("阿斯顿789")
	fmt.Println(strings.ToUpper(abc))
}

func Test_CheckMobileNumber(t *testing.T) {
	abc := CheckMobileNumber("12345678910")
	fmt.Println(abc)
}
