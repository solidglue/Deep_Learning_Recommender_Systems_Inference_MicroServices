package nacos_config_listener

import (
	"infer-microservices/common/flags"
	"infer-microservices/cores/service_config_loader"
	"infer-microservices/utils/logs"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var nacosTimeoutMs uint64
var nacosLogDir string
var nacosCacheDir string
var nacosLogLevel string
var nacosUsername string
var nacosPassword string
var mt sync.Mutex
var NacosListedMap = make(map[string]bool, 0)

type NacosConnConfig struct {
	dataId      string
	groupId     string
	namespaceId string
	ip          string
	port        uint64
}

func init() {
	flagFactory := flags.FlagFactory{}
	flagNacos := flagFactory.CreateFlagNacos()

	nacosTimeoutMs = uint64(*flagNacos.GetNacosTimeoutMs())
	nacosLogDir = *flagNacos.GetNacosLogdir()
	nacosCacheDir = *flagNacos.GetacosCachedir()
	nacosLogLevel = *flagNacos.GetNacosLoglevel()
	nacosUsername = *flagNacos.GetNacosUsername()
	nacosPassword = *flagNacos.GetNacosPassword()
}

// dataId
func (r *NacosConnConfig) SetDataId(dataId string) {
	r.dataId = dataId
}

func (r *NacosConnConfig) GetDataId() string {
	return r.dataId
}

// groupId
func (r *NacosConnConfig) SetGroupId(groupId string) {
	r.groupId = groupId
}

func (r *NacosConnConfig) GetGroupId() string {
	return r.groupId
}

// namespaceId
func (r *NacosConnConfig) SetNamespaceId(namespaceId string) {
	r.namespaceId = namespaceId
}

func (r *NacosConnConfig) GetNamespaceId() string {
	return r.namespaceId
}

// ip
func (r *NacosConnConfig) SetIp(ip string) {
	r.ip = ip
}

func (r *NacosConnConfig) GetIp() string {
	return r.ip
}

// port
func (r *NacosConnConfig) SetPort(port uint64) {
	r.port = port
}

func (r *NacosConnConfig) GetPort() uint64 {
	return r.port
}

func (n *NacosConnConfig) ServiceConfigListen() error {

	nacosClient, err := n.getNacosClient()
	if err != nil {
		return err
	}

	err = n.listenNacosConfig(nacosClient)
	if err != nil {
		return err
	}

	return nil
}

func (n *NacosConnConfig) getNacosClient() (config_client.IConfigClient, error) {
	serviceConf := []constant.ServerConfig{{
		IpAddr: n.GetIp(),
		Port:   n.GetPort(),
	}}

	clientConf := constant.ClientConfig{
		NamespaceId:         n.GetNamespaceId(),
		TimeoutMs:           nacosTimeoutMs,
		NotLoadCacheAtStart: true,
		LogDir:              nacosLogDir,
		CacheDir:            nacosCacheDir,
		LogLevel:            nacosLogLevel,
		Username:            nacosUsername,
		Password:            nacosPassword,
	}
	nacosClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serviceConf,
		"clientConfig":  clientConf,
	})
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	return nacosClient, nil
}

func (n *NacosConnConfig) getNacosConfig(nacosClient config_client.IConfigClient) (string, error) {
	//TODO: VERIFY CONFIG JSON
	content, err := nacosClient.GetConfig(vo.ConfigParam{
		DataId: n.GetDataId(),
		Group:  n.GetGroupId(),
	})
	if err != nil {
		return "", err
	}
	return content, nil
}

func (n *NacosConnConfig) listenNacosConfig(nacosClient config_client.IConfigClient) error {
	err := nacosClient.ListenConfig(vo.ConfigParam{
		DataId: n.GetDataId(),
		Group:  n.GetGroupId(),
		OnChange: func(namespace, group, dataId, data string) {
			content := string(data)
			n.serviceConfigUpdate(dataId, content)
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (n *NacosConnConfig) serviceConfigUpdate(dataId string, content string) error {
	mt.Lock()
	defer mt.Unlock()

	builder := service_config_loader.ServiceConfigBuilder{}
	director := service_config_loader.ServiceConfigDirector{}
	director.SetConfigBuilder(builder)

	nacosContent := nacosContent{}
	domain, redisConfStr, modelConfStr, indexConfStr := nacosContent.InputServiceConfigParse(content)

	var serviceConf service_config_loader.ServiceConfig
	if indexConfStr == "{}" {
		//recall
		serviceConf = director.ServiceConfigUpdateContainIndexDirector(domain, dataId, redisConfStr, modelConfStr, indexConfStr)
	} else {
		//rank
		serviceConf = director.ServiceConfigUpdaterNotContainIndexDirector(domain, dataId, redisConfStr, modelConfStr)
	}

	service_config_loader.ServiceConfigs[dataId] = &serviceConf

	return nil
}
