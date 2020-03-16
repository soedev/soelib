package soelog

/**
  soelog  公共日志类
*/
import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//Logger 日志
var Logger *zap.Logger

var (
	//LogSavePath 日志保存路径
	LogSavePath = "logs/"
	//LogSaveName 日志文件名
	LogSaveName = "log"
	//LogFileExt 日志扩展名
	LogFileExt = "log"
	//TimeFormat 文件名格式
	TimeFormat = "20060102"
)

func getLogFilePath() string {
	return fmt.Sprintf("%s", LogSavePath)
}

//GetLogFileFullPath 获取日志文件路径
func GetLogFileFullPath() string {
	prefixPath := getLogFilePath()
	suffixPath := fmt.Sprintf("%s%s.%s", LogSaveName, time.Now().Format(TimeFormat), LogFileExt)

	return fmt.Sprintf("%s%s", prefixPath, suffixPath)
}

func mkDir(filePath string) {
	dir, _ := os.Getwd()
	err := os.MkdirAll(dir+"/"+getLogFilePath(), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

//InitLogger 初始化日志
func InitLogger(isDebug bool) {
	// 检测文件夹是否存在
	filePath := GetLogFileFullPath()
	_, err := os.Stat(filePath)
	switch {
	case os.IsNotExist(err):
		mkDir(getLogFilePath())
	case os.IsPermission(err):
		log.Fatalf("Permission :%v", err)
	}

	// 日志地址 "out.log" 自定义
	lp := GetLogFileFullPath()
	// 日志级别 DEBUG,ERROR, INFO
	lv := "INFO" //Conf.Common.LogLevel
	// 是否 DEBUG
	//if Conf.Common.IsDebug != true {
	//	isDebug = false
	//}
	initLogger(lp, lv, isDebug)
	//log.SetFlags(log.Lmicroseconds | log.Lshortfile | log.LstdFlags)
}

func initLogger(lp string, lv string, isDebug bool) {
	var js string
	if isDebug {
		js = fmt.Sprintf(`{
      "level": "%s",
      "encoding": "console",
      "outputPaths": ["stdout"],
      "errorOutputPaths": ["stdout"]
      }`, lv)
	} else {
		js = fmt.Sprintf(`{
      "level": "%s",
      "encoding": "console",
      "outputPaths": ["%s"],
      "errorOutputPaths": ["%s"]
      }`, lv, lp, lp)
	}

	//"encoding": "json"   //json格式输出

	var cfg zap.Config
	if err := json.Unmarshal([]byte(js), &cfg); err != nil {
		panic(err)
	}
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	Logger, err = cfg.Build()
	if err != nil {
		log.Fatal("init logger error: ", err)
	}
}

//GetLastLines 取最后几行日志
func GetLastLines(lines int64) string {

	//lines := int64(200) //读最后200行
	file, err := os.Open(GetLogFileFullPath())
	if err != nil {
		log.Println(err)
		return ""
	}
	fileInfo, _ := file.Stat()
	buf := bufio.NewReader(file)
	offset := fileInfo.Size() % 8192
	data := make([]byte, 8192) // 一行的数据
	totalByte := make([][][]byte, 0)
	readLines := int64(0)
	for i := int64(0); i <= fileInfo.Size()/8192; i++ {
		readByte := make([][]byte, 0) // 读取一页的数据
		file.Seek(fileInfo.Size()-offset-8192*i, io.SeekStart)
		data = make([]byte, 8192)
		n, err := buf.Read(data)
		if err == io.EOF {
			if strings.TrimSpace(string(bytes.Trim(data, "\x00"))) != "" {
				readLines++
				readByte = append(readByte, data)
				totalByte = append(totalByte, readByte)
			}
			if readLines > lines {
				break
			}
			continue
		}
		if err != nil {
			log.Println("Read file error:", err)
			return ""
		}
		strs := strings.Split(string(data[:n]), "\n")
		if len(strs) == 1 {
			b := bytes.Trim([]byte(strs[0]), "\x00")
			if len(b) == 0 {
				continue
			}
		}
		if (readLines + int64(len(strs))) > lines {
			strs = strs[int64(len(strs))-lines+readLines:]
		}
		for j := 0; j < len(strs); j++ {
			readByte = append(readByte, bytes.Trim([]byte(strs[j]+"\n"), "\x00"))
		}
		readByte[len(readByte)-1] = bytes.TrimSuffix(readByte[len(readByte)-1], []byte("\n"))
		totalByte = append(totalByte, readByte)
		readLines += int64(len(strs))

		if readLines >= lines {
			break
		}
	}
	totalByte = ReverseByteArray(totalByte)
	return ByteArrayToString(totalByte)
}

//ReverseByteArray ReverseByteArray
func ReverseByteArray(s [][][]byte) [][][]byte {
	for from, to := 0, len(s)-1; from < to; from, to = from+1, to-1 {
		s[from], s[to] = s[to], s[from]
	}
	return s
}

//ByteArrayToString ByteArrayToString
func ByteArrayToString(buf [][][]byte) string {
	str := make([]string, 0)
	for _, v := range buf {
		for _, vv := range v {
			str = append(str, string(vv))
		}
	}
	return strings.Join(str, "")
}

//ClearLogsBeforeDay 清空几天前的日志文件
func ClearLogsBeforeDay() {
	files, _ := ioutil.ReadDir(getLogFilePath())
	for _, fi := range files {
		if fi.IsDir() {
			//文件夹不处理
		} else {
			//删除三天前的日志文件
			d, _ := time.ParseDuration("-72h")
			twoDaysAgo := time.Now().Add(d)
			if fi.ModTime().Before(twoDaysAgo) {
				os.Remove(getLogFilePath() + fi.Name())
			}
			//log.Println(fi.Name() + fi.ModTime().Format("2006-01-02 15:04:05.999")) //.Before(time.Now())
		}
	}
}
