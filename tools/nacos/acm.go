package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type AcmConfig struct {
	Endpoint    string
	NamespaceID string
	AccessKey   string
	SecretKey   string
}

var AcmClient *config_client.IConfigClient

//InitAcm 初始化acm
func InitAcm(acmConfig AcmConfig) (err error) {
	acmClient, err := clients.CreateConfigClient(map[string]interface{}{
		"clientConfig": constant.ClientConfig{
			Endpoint:    acmConfig.Endpoint,
			NamespaceId: acmConfig.NamespaceID,
			AccessKey:   acmConfig.AccessKey,
			SecretKey:   acmConfig.SecretKey,
			TimeoutMs:   5 * 1000,
		},
	})
	AcmClient = &acmClient
	if err != nil {
		return err
	}
	return nil
}

//GetAcmContent 获取配置
func GetAcmContent(dataID, group string) (content string, err error) {
	acmClient := *AcmClient
	// 获取配置
	content, err = acmClient.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group})
	if err != nil {
		return "", err
	}
	return content, err
}
