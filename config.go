package network

import "github.com/lengzhao/network/log"

// NetworkConfig 网络配置
type NetworkConfig struct {
	Host           string         // 监听地址
	Port           int            // 监听端口
	MaxPeers       int            // 最大连接数
	PrivateKeyPath string         // 私钥文件路径
	BootstrapPeers []string       // Bootstrap节点地址列表
	PeerWhitelist  []string       // 节点白名单(节点ID字符串列表)
	LogConfig      *log.LogConfig // 日志配置
	DisableMDNS    bool           // 是否禁用MDNS发现功能，默认为false（即默认启用MDNS）
	DisableDHT     bool           // 禁用dht
}

// NewNetworkConfig 创建一个新的网络配置实例，使用默认值
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		Host:        "0.0.0.0",
		Port:        0,
		MaxPeers:    100,
		DisableMDNS: false, // 默认不禁用MDNS，即默认启用MDNS
	}
}
