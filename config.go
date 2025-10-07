package network

import "github.com/lengzhao/network/log"

// NetworkConfig 网络配置
type NetworkConfig struct {
	Host           string   // 监听地址
	Port           int      // 监听端口
	MaxPeers       int      // 最大连接数
	PrivateKeyPath string   // 私钥文件路径
	BootstrapPeers []string // Bootstrap节点地址列表
	PeerWhitelist  []string // 节点白名单(节点ID字符串列表)
	LogConfig      *log.LogConfig // 日志配置
}