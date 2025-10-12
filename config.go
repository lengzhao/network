package network

// NetworkConfig 网络配置
type NetworkConfig struct {
	Host           string   // 监听地址
	Port           int      // 监听端口
	MaxPeers       int      // 最大连接数
	PrivateKeyPath string   // 私钥文件路径
	BootstrapPeers []string // Bootstrap节点地址列表
	PeerWhitelist  []string // 节点白名单(节点ID字符串列表)
	DisableMDNS    bool     // 是否禁用MDNS发现功能

	// 新增的配置字段
	EnablePeerScoring  bool    // 是否启用Peer评分
	MaxIPColocation    int     // 单个IP地址最大节点数
	IPColocationWeight float64 // IP共置权重
	BehaviourWeight    float64 // 行为权重
	BehaviourDecay     float64 // 行为衰减因子
	AppSpecificScore   float64 // 应用特定评分
}

// NewNetworkConfig 创建一个新的网络配置实例，使用默认值
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		Host:     "0.0.0.0",
		Port:     0,
		MaxPeers: 100,

		// 默认启用Peer评分
		EnablePeerScoring:  true,
		MaxIPColocation:    3,    // 单个IP地址最多3个节点
		IPColocationWeight: -0.1, // IP共置权重
		BehaviourWeight:    -1.0, // 行为权重
		BehaviourDecay:     0.98, // 行为衰减因子
	}
}
