package soetcp

/**
  小索辅助消息实体类
*/
import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/google/uuid"
	"github.com/soedev/soelib/common/utils"
	"strings"
)

const (
	CommandExit                   = 0
	CommandTTS                    = 1
	CommandTeaPrint               = 2 //茶水打印
	CommandRoomCall               = 3 //房间呼叫
	CommandDepurate               = 4 //打扫呼叫
	CommandNamedAddClock          = 5 //加钟点钟
	CommandEndDepurate            = 6 //结束打扫
	CommandEndClockNoWorker       = 7 //
	CommandPrintWorkerGoods       = 8
	CommandTTSData                = 9
	CommandQuestion               = 10
	CommandAnswer                 = 11
	CommandChangeRoom             = 12 //换房
	CommandClientList             = 13
	CommandCheckOut               = 14 //结帐
	CommandSatisfy                = 15 //满意卡
	CommandXunLingRemindData      = 16 //讯铃
	CommandSmartRemindData        = 17
	CommandDestine                = 18 //预约
	CommandDelGoods               = 19 //退项目
	CommandTeaAwokeToSpeak        = 20 //点茶水提醒
	CommandCustom                 = 21 //自定义
	CommandAutoRemindScheme       = 22 //自动提醒方案
	CommandCleanWater             = 23 //
	CommandConsumeSpicePrint      = 24 //药水
	CommandFastInvoiceCheckout    = 25
	CommandFastInvoiceReceiveOpen = 26 //快速开单_已开单
	CommandOnlineRoomCall         = 27 //线上呼叫类型（lotus）
	CommandOnlineConsume          = 28
	CommandOnlineRoomCallPrint    = 29 //房间呼叫打印 [HTTP]
	CommandOnlinePayPrint         = 30 //在线支付打印 [HTTP]
	CommandMainPromData           = 31
)

//SRTContent 呼叫播报XML
type SRTContent struct {
	XMLName       xml.Name `xml:"SmartRemindData"`
	ForwardRoomID string   `xml:"ForwardRoomID" json:"rorwardRoomId"` //包厢主键
	RoomID        string   `xml:"RoomID" json:"roomId"`
	WorkerID      string   `xml:"WorkerID" json:"workerId"` //员工主键
	Value         string   `xml:"Value" json:"value"`
	RemindType    string   `xml:"RemindType" json:"remindType"` //播报类型
	GoodsName     string   `xml:"GoodsName" json:"goodsName"`
	Amount        string   `xml:"Amount" json:"amount"`
	Print         string   `xml:"printer" json:"printer"`
	Remark        string   `xml:"remark" json:"remark"`
}

//TeaDataContent 呼叫茶水XML
type TeaDataContent struct {
	XMLName     xml.Name `xml:"TeaData"`
	RoomID      string   `xml:"RoomID"`
	HandBrandID string   `xml:"HandBrandID"`
	GoodsName   string   `xml:"GoodsName"`
	Amount      float32  `xml:"Amount"`
	Printer     string   `xml:"Printer"`
	WorkerID    string   `xml:"WorkerID"`
	Remark      string   `xml:"Remark"`
}

//DirectiveContent 发送指令内容
type DirectiveContent struct {
	Command        int
	CommandType    string
	WorkerID       string
	ForwardRoomID  string
	Value          string
	Type           string
	Computer       string `xml:"Computer"`       //电脑
	Printer        string `xml:"Printer"`        //打印机名称多个打印机使用｜线分隔
	SoundDevice    string `xml:"SoundDevice"`    //声卡
	PlaySyc        int    `xml:"PlaySyc"`        //声音遍数或打印份数
	PlayType       int    `xml:"PlayType"`       //0.无操作　1.提示音　2.语音　3.提示音+语音
	IsNewInterface int    `xml:"IsNewInterface"` //0.无操作　1.提示音　2.语音　3.提示音+语音
}

//服务通讯类
type SoeServiceData struct {
	XMLName        xml.Name `xml:"SoeServiceData,omitempty"`
	Command        int      `xml:"Command" json:"command"`               //命令类型
	Content        string   `xml:"Content" json:"content"`               //发送内容
	Source         int      `xml:"Source" json:"source"`                 //来源
	GuID           string   `xml:"Guid" json:"guid"`                     //唯一码
	Computer       string   `xml:"Computer" json:"computer"`             //电脑
	Printer        string   `xml:"Printer" json:"printer"`               //打印机名称多个打印机使用｜线分隔
	SoundDevice    string   `xml:"SoundDevice" json:"soundDevice"`       //声卡
	PlaySyc        int      `xml:"PlaySyc" json:"playSyc"`               //声音遍数或打印份数
	PlayType       int      `xml:"PlayType" json:"playType"`             //0.无操作　1.提示音　2.语音　3.提示音+语音
	IsNewInterface int      `xml:"IsNewInterface" json:"isNewInterface"` //0.无操作　1.提示音　2.语音　3.提示音+语音
}

func (s *SoeServiceData) toXML() (xmlContent string, err error) {
	uid := uuid.New().String()
	s.GuID = "{" + strings.ToUpper(uid) + "}"
	s.Content = base64.StdEncoding.EncodeToString([]byte(s.Content))
	data, _ := xml.MarshalIndent(s, "", " ")
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>` + string(data)
	if encoderData, isOk := utils.GBKEncoder(xmlData); isOk {
		xmlData = encoderData
	}
	return xmlData, nil
}

//发送数据到小索辅助
func SendServiceData(datas []SoeServiceData) {
	for _, v := range datas {
		xml, _ := v.toXML()
		err := SendMessage(xml)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

//GenerateSRTContentJSON 生成相应播报类型内容
func GenerateSRTContentJSON(content SRTContent) string {
	contentData, _ := json.Marshal(content)
	return string(contentData)
}

//GenerateDirectiveContentJSON 生成相应播报类型内容
func GenerateDirectiveContentJSON(content DirectiveContent) string {
	contentData, _ := json.Marshal(content)
	return string(contentData)
}

//GenerateSRTWaringJSON 生成智钟宝语音播报
func GenerateSRTWaringJSON(content SRTContent) string {
	contentStr := GenerateSRTContentJSON(content)
	uid := uuid.New().String()
	soeServiceData := SoeServiceData{Command: CommandSmartRemindData, Content: contentStr, Source: 1,
		GuID: "{" + strings.ToUpper(uid) + "}", PlaySyc: 0, PlayType: 0, IsNewInterface: 0}
	contentData, _ := json.Marshal(soeServiceData)
	return string(contentData)
}

//GenerateVoiceJSON 生成语音播报XML
func GenerateVoiceJSON(content DirectiveContent) (xmlContent string, err error) {
	uid := uuid.New().String()
	soeServiceData := SoeServiceData{Command: content.Command, Content: content.Value, Source: 0,
		GuID: "{" + strings.ToUpper(uid) + "}", PlaySyc: content.PlaySyc, Computer: content.Computer,
		SoundDevice: content.SoundDevice, PlayType: content.PlayType, IsNewInterface: 1}
	contentData, _ := json.Marshal(soeServiceData)
	return string(contentData), nil
}

//GenerateLotusVoiceJSON 生成Lotus调用呼叫
func GenerateLotusVoiceJSON(content SRTContent, serviceCommand int) string {
	contentStr := GenerateSRTContentJSON(content)
	uid := uuid.New().String()
	soeServiceData := SoeServiceData{Command: serviceCommand, Content: contentStr, Source: 5,
		GuID: "{" + strings.ToUpper(uid) + "}", PlaySyc: 0,
		PlayType: 0, IsNewInterface: 0}
	contentData, _ := json.Marshal(soeServiceData)
	return string(contentData)
}

//GenerateSRTContentXML 生成相应播报类型内容xml
func GenerateSRTContentXML(command interface{}) string {
	contentData, _ := xml.MarshalIndent(&command, "", " ")
	contentStr := `<?xml version="1.0" encoding="UTF-8"?>` + string(contentData)
	return contentStr
}

//GenerateSRTWaringXML 生成智钟宝语音播报xml
func GenerateSRTWaringXML(command SRTContent) string {
	contentStr := GenerateSRTContentXML(command)
	contentBase64 := base64.StdEncoding.EncodeToString([]byte(contentStr))
	uid := uuid.New()
	ssd := SoeServiceData{Command: CommandSmartRemindData, Content: string(contentBase64), Source: 1,
		GuID: "{" + strings.ToUpper(uid.String()) + "}", PlaySyc: 1, PlayType: 0, IsNewInterface: 1}
	data, _ := xml.MarshalIndent(&ssd, "", " ")
	return `<?xml version="1.0" encoding="UTF-8"?>` + string(data)
}

//GenerateVoiceXML 生成语音播报XML
func GenerateVoiceXML(param DirectiveContent) (xmlContent string, err error) {
	contentBase64 := base64.StdEncoding.EncodeToString([]byte(param.Value))
	uid := uuid.New()
	ssd := SoeServiceData{Command: param.Command, Content: string(contentBase64), Source: 0,
		GuID: "{" + strings.ToUpper(uid.String()) + "}", PlaySyc: 1, Computer: param.Computer,
		SoundDevice: param.SoundDevice, PlayType: param.PlayType, IsNewInterface: 1}
	data, _ := xml.MarshalIndent(&ssd, "", " ")
	return `<?xml version="1.0" encoding="UTF-8"?>` + string(data), nil
}

//GenerateLotusVoiceXML 生成Lotus调用呼叫
func GenerateLotusVoiceXML(command SRTContent, serviceCommand int) string {
	contentStr := GenerateSRTContentXML(command)
	contentBase64 := base64.StdEncoding.EncodeToString([]byte(contentStr))
	uid := uuid.New()
	ssd := SoeServiceData{Command: serviceCommand, Content: string(contentBase64), Source: 5,
		GuID: "{" + strings.ToUpper(uid.String()) + "}", PlaySyc: 0,
		PlayType: 0, IsNewInterface: 0}
	data, _ := xml.MarshalIndent(&ssd, "", " ")
	return `<?xml version="1.0" encoding="UTF-8"?>` + string(data)
}
