package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"
)

//ChatMsg 企业微信消费
type ChatMsg struct {
	ChatID  string `json:"chatID"`
	Content string `json:"content"`
}

//DefaultRegChatID 插件注册预警群  16613216078422402691
const DefaultRegChatID = "16613216078422402691"

//WorkWxAPIPath 企业微信rest路径
const WorkWxAPIPath = "https://www.soesoft.org/workwx-rest/api/send-msg-to-chat"

// WorkWxRestTokenStr 企业微信访问token
const WorkWxRestTokenStr = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4NDU3MTYyNzUsImlzcyI6InBvd2VyIiwic3ViIjoie1wiVXNlclVJRFwiOlwiXCIsXCJUZW5hbnRJRFwiOlwiXCIsXCJUZW5hbnRDb2RlXCI6XCJzb2Vzb2Z0XCIsXCJBbGlVc2VyUElEXCI6XCJcIixcIkFsaU1lcmNoYW50UElEXCI6XCJcIixcIkFsaUF1dGhUb2tlblwiOlwiXCIsXCJBbGlBdXRoQ29kZVwiOlwiXCIsXCJPcGVuSURcIjpcIlwiLFwiT3BlbklEMlwiOlwiXCIsXCJMb2dpblR5cGVcIjpcIlwiLFwiSG9sZFNob3BDb2RlXCI6XCJcIixcIkFsaU1lcmNoYW50U2hvcElEXCI6XCJcIn0ifQ.TZccfoDPMFFrZm5nvojaeXiXnEpxKloM5IkdQB2rTBg"

// SendMsgToWorkWx 发送信息到企业微信会话
func SendMsgToWorkWx(chatid, content, apiPath, tokenStr string) error {
	/*
		body := fmt.Sprintf(`
		{
			"chatID":"%s",
			"content":"%s"
		}`, chatid, content)
	*/
	chatMsg := ChatMsg{
		ChatID:  chatid,
		Content: content,
	}
	postBody, _ := json.Marshal(chatMsg)

	tr := &http.Transport{ //解决x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr, //解决x509: certificate signed by unknown authority
	}

	//req, err := http.NewRequest("POST", apiPath, bytes.NewReader([]byte(body)))
	req, err := http.NewRequest("POST", apiPath, bytes.NewReader(postBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Authorization", tokenStr)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
