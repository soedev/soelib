package nacos

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	configClient config_client.IConfigClient
	namingClient naming_client.INamingClient
	group        string
	namespace    string
	accessKey    string
	secretKey    string
}

//通知回调
type OnChange func(content, dataID string)

var NacosClient *Client

func NewNacosClient(nodes []string, config Config) (err error) {
	var configClient config_client.IConfigClient
	servers := []constant.ServerConfig{}
	for _, key := range nodes {
		nacosUrl, _ := url.Parse(key)

		port, _ := strconv.Atoi(nacosUrl.Port())
		servers = append(servers, constant.ServerConfig{
			IpAddr: nacosUrl.Hostname(),
			Port:   uint64(port),
		})
	}

	if len(strings.TrimSpace(config.Group)) == 0 {
		config.Group = "DEFAULT_GROUP"
	}

	fmt.Println(fmt.Sprintf("endpoint=%s, namespace=%s, group=%s, accessKey=%s, secretKey=%s, openKMS=%d, regionId=%s", config.Endpoint, config.Namespace, config.Group, config.AccessKey, config.SecretKey, config.OpenKMS, config.RegionId))

	configClient, err = clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": servers,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           10000,
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			NamespaceId:         config.Namespace,
			AccessKey:           config.AccessKey,
			SecretKey:           config.SecretKey,
			Endpoint:            config.Endpoint,
			OpenKMS:             config.OpenKMS,
			RegionId:            config.RegionId,
		},
	})

	namingClient, _ := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": servers,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           10000,
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			NamespaceId:         config.Namespace,
			AccessKey:           config.AccessKey,
			SecretKey:           config.SecretKey,
			Endpoint:            config.Endpoint,
		},
	})
	NacosClient = &Client{configClient, namingClient, config.Group, config.Namespace, config.AccessKey, config.SecretKey}
	return
}

func (client *Client) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, key := range keys {
		if strings.HasPrefix(key, "naming.") {
			instances, err := client.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
				ServiceName: key,
				GroupName:   client.group,
				//HealthyOnly: true,
			})

			fmt.Println(fmt.Sprintf("key=%s, value=%s", key, instances))
			if err == nil {
				vars[key] = util.ToJsonString(instances)
			}
		} else {
			resp, err := client.configClient.GetConfig(vo.ConfigParam{
				DataId: key,
				Group:  client.group,
			})
			fmt.Println(fmt.Sprintf("key=%s, value=%s", key, resp))

			if err == nil {
				vars[key] = resp
			} else {
				fmt.Println(fmt.Sprintf("getv key=%s  err=%s", key, err.Error()))
			}
		}
	}

	return vars, nil
}

func (client *Client) WatchPrefix(keys []string, call OnChange) error {
	for _, key := range keys {
		if strings.HasPrefix(key, "naming.") {
			client.namingClient.Subscribe(&vo.SubscribeParam{
				ServiceName: key,
				GroupName:   client.group,
				SubscribeCallback: func(services []model.SubscribeService, err error) {
					fmt.Println(fmt.Sprintf("\n\n callback return services:%s \n\n", util.ToJsonString(services)))
					call("", key)
				},
			})
		} else {
			err := client.configClient.ListenConfig(vo.ConfigParam{
				DataId: key,
				Group:  client.group,
				OnChange: func(namespace, group, dataId, data string) {
					fmt.Println(fmt.Sprintf("config namespace=%s, dataId=%s, group=%s has changed", namespace, dataId, group))
					call(data, key)
				},
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
