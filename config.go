package network

// NetworkConfig 网络配置
type NetworkConfig struct {
	Host           string   // 监听地址
	Port           int      // 监听端口
	MaxPeers       int      // 最大连接数
	PrivateKeyPath string   // 私钥文件路径
	BootstrapPeers []string // Bootstrap节点地址列表
}

// NewNetworkConfig 创建一个新的网络配置实例，使用默认值
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		Host:     "0.0.0.0",
		Port:     0,
		MaxPeers: 100,
	}
}
