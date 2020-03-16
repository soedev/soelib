package utils

/**
  file   文件处理工具类
*/

import (
	"encoding/json"
	"os"
	"path"
)

//PathExists 检查路径是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

//LoadJsonConfig  读取json 格式的配置文件 转成对象
func LoadJsonConfig(configFile string, v interface{}) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	return decoder.Decode(v)
}

//WriteJsonConfig 把json 对象 写入硬盘文件
func WriteJsonConfig(filePath string, jsonByte []byte) error {
	_, err := writeBytes(filePath, jsonByte)
	return err
}

func writeBytes(filePath string, b []byte) (int, error) {
	os.MkdirAll(path.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}
