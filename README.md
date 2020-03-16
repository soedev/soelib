# github.com/soedev/soelib
soe go服务公共工具包

github.com/soedev/soelib/common/utils   //常用公共方法
github.com/soedev/soelib/common/net     //http请求,emqtt
github.com/soedev/soelib/common/tools   //连接池,定时任务,协程池,插槽消息


使用方式:
在go.mod 中配置
require github.com/soedev/soelib 版本号 拉取

注意:当更新soelib公共包代码时,go项目需要通过命令 go get -u github.com/soedev/soelib 升级依赖到最新版本 
