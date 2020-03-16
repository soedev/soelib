package register

/**
  update  软件更新业务类
*/

import (
	"github.com/inconshreveable/go-update"
	"log"
	"net/http"
	"time"
)

//UpdateSelf 程序自升级  更新时间、当前版本、服务器版本、更新地址URL
func UpdateSelf(updateDate *time.Time, curVersion, updateVer, updateURL string) bool {
	now := time.Now()
	updateNow := false
	//判断是否现在升级
	if updateDate == nil {
		updateNow = true
	} else {
		//解决时间相差 8 小时问题
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", updateDate.String(), time.Local)
		if now.After(t) {
			updateNow = true
		}
	}
	if updateVer != "" && updateVer != curVersion && updateNow {
		err := doUpdate(updateVer, updateURL)
		return err == nil
	}
	return false
}

//DoUpdate 开始更新
func doUpdate(ver, url string) error {
	log.Printf("开始自动更新 版本:%s 路径:%s", ver, url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("下载自动更新失败:%s", err.Error())
		return err
	}
	defer resp.Body.Close()
	err = update.Apply(resp.Body, update.Options{TargetMode: 0775})
	if err != nil {
		return err
	}
	return nil
}
