package soetcp

/**
  soetcp  小索辅助连接类
*/

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"time"
)

//消息反馈 函数
type TcpCall func(callMsg string)

//Conn TCP连接
var conn net.Conn

var msgCall TcpCall

type TcpConfig struct {
	Host string //小索辅助服务器IP
	Port string //服务器端口
	Type string //客户端类型【用来在客户端展示列表中显示出来】
}

//NewClient 新建TCP连接  IP 端口 程序识别类型 消息反馈
func NewClient(config TcpConfig, call TcpCall) {
	msgCall = call
	hostInfo := config.Host + ":" + config.Port
	go ReconnectionService(hostInfo, config.Type)
}
func NewClientWithCall(host, port, pType string) {
	msgCall = nil
	hostInfo := host + ":" + port
	go ReconnectionService(hostInfo, pType)
}

//ReconnectionService 建立TCP连接及重连
func ReconnectionService(hostInfo, pType string) {
	for {
		conn, err := net.Dial("tcp", hostInfo)
		if err == nil {
			showLog(fmt.Sprintf("连接 %s 小索", hostInfo))
			doTask(conn, pType)
		}
		time.Sleep(3 * time.Second)
	}
}

//doTask 重连TCP
func doTask(connTemp net.Conn, pType string) {
	conn = connTemp
	for {
		msg, err := bufio.NewReader(connTemp).ReadString('\n')
		if err != nil {
			showLog("小索辅助服务器已断开")
			break
		}
		if msg == "WHOISSOE\r\n" { //首次连接上,发送自己的设备信息
			SendMessage(pType)
		}
		time.Sleep(1 * time.Second)
	}
	conn = nil
	defer connTemp.Close()
}

//SendMessage TCP通信 发送命令到小索辅助
func SendMessage(message string) (err error) {
	if conn == nil {
		return errors.New("未连接上小索辅助")
	}
	n, err := conn.Write([]byte(message + "\r\n"))
	showLog(fmt.Sprintf("SEND：%d", n))
	if err != nil {
		return err
	}
	//接收服务端反馈
	go ReadResult()
	return nil
}

//ReadResult 接收服务端返回的数据
func ReadResult() {
	result, err := ioutil.ReadAll(conn)
	if err != nil {
		showLog(err.Error())
		return
	}
	showLog("小索通信服务端反馈:" + string(result))
}

func showLog(msg string) {
	if msgCall != nil {
		msgCall(msg)
	}
}
