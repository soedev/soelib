package register

/**
  索易软件注册业务类
*/

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/soedev/soelib/common/des"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

//Reg 注册客户端信息
type Reg struct {
	Kind       string //软件类型：主程序，智钟宝
	PlugInName string //软件，手持pos 从插件中校验到期日期
	FileName   string //全路径文件名
	Ver        string //当前客户端版本号，需要判断是否要强制升级
	TenantCode string //从系统参数中取出
	TenantID   int
	Token      string `binding:"required"` //deviceID + date   des加密
	DeviceID   string `binding:"required"` //序列号   必填字段
	OsInfo     OsInfo //操作系统信息
	IP         string //外网ip地址，自动获取，不用客户端填入
}

//RegResponse 信息返回结构体
type RegResponse struct {
	Code int     `json:"code"`
	Msg  string  `json:"msg"`
	Data RegInfo `json:"data"`
}

//RegInfo 校验返回信息
type RegInfo struct {
	Company        string     `json:"company"`       //商户名
	ShopID         string     `json:"shopID"`        //店号
	ShopCode       string     `json:"shopCode"`      //店唯一编号
	ShopName       string     `json:"shopName"`      //店名
	TenantID       int        `json:"tenantID"`      //租户信息
	TenantCode     string     `json:"tenantCode"`    //租户信息
	Message        string     `json:"message"`       //提示信息，比如可以发公司公告
	URL            string     `json:"url"`           //信息网址
	DeviceID       string     `json:"deviceID"`      //传入的序列号
	ExpiredDate    string     `json:"expiredDate"`   //到期日期
	PayURL         string     `json:"payURL"`        //续费网址
	UpdateVer      string     `json:"updateVer"`     //服务端版本号，用于校验是否要升级
	UpdateURL      string     `json:"updateURL"`     //升级版本路径
	UpdateMessage  string     `json:"updateMessage"` //升级版本信息
	UpdateDate     *time.Time `json:"updateDate"`    //升级日期
	CheckSum       string     `json:"checkSum"`
	TerminalName   string     `json:"terminalName"`   //终端名     主机，收银，客户端
	Token          string     `json:"token"`          //token
	JSONInfo       string     `json:"jsonInfo"`       //其它预留信息
	OperationValue string     `json:"operationValue"` //注册功能码
}

type OsInfo struct {
	OS            string
	LocalIP       string //内网ip，是否自动获取
	MemerySize    string //内存大小
	CPU           string
	DiskSpace     string //软件所在磁盘大小
	FreeDiskSpace string //剩余磁盘空间
}

const (
	plugInName = "软件"
)

//CheckReg 软件注册 是否调试、租户号、软件版本、注册唯一编码、软件kind、强制店号检测（码重复的时候，本地设置店号用来强制检测）
func CheckReg(regURL string, tenantID int, version, deviceID, kind, checkShop string) (*RegInfo, error) {
	reg := Reg{
		Kind:       kind,
		PlugInName: plugInName,
		Ver:        version,
		TenantID: tenantID,
		DeviceID: deviceID,
	}
	return CheckRegWithOsInfo(reg,regURL,checkShop)
}

//CheckRegWithOsInfo 软件注册 是否调试、租户号、软件版本、注册唯一编码、软件kind、强制店号检测（码重复的时候，本地设置店号用来强制检测）
func CheckRegWithOsInfo(reg Reg,regURL,checkShop string) (*RegInfo, error) {
	var regResponse = RegResponse{}
	filePath, _ := os.Executable()
	powerDes := des.PowerDes{}
	token, _ := powerDes.PowerEncryStr(reg.DeviceID+time.Now().Format("2006-01-02"), des.PowerDesKey)
	reg.FileName = filePath
	reg.Token = token

	postBody, _ := json.Marshal(reg)

	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}
	req, err := http.NewRequest("POST", regURL, bytes.NewReader(postBody))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用注册接口时出错:%s", err.Error()))
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用注册接口时取数据出错:%s", err.Error()))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用注册接口时ReadAll取数据出错:%s", err.Error()))
	}
	err = json.Unmarshal(body, &regResponse)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("调用注册接口时取数据出错:%s", err.Error()))
	}
	defer resp.Body.Close()
	if regResponse.Code != 0 {
		return nil, errors.New(fmt.Sprintf("校验注册失败:%s", regResponse.Msg))
	}
	if checkShop != "" && checkShop != regResponse.Data.ShopID {
		return nil, errors.New(fmt.Sprintf("本地存在分店与线上不一致 本地：%s  实际：%s", checkShop, regResponse.Data.ShopID))
	}
	return &regResponse.Data, nil
}