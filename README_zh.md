# 基于go-libp2p的通用网络模块

## 概述

这是一个基于go-libp2p的通用网络模块，可用于区块链、聊天系统等多种P2P应用场景。该模块提供简单易用的API，支持广播模式和点对点请求模式，并为广播模式提供过滤功能。

## 功能特性

- **双重通信模式**：支持广播模式（发布/订阅）和点对点请求/响应模式
- **简单易用**：提供简洁的API接口，降低P2P网络编程的复杂性
- **消息过滤**：广播模式支持消息过滤功能
- **自动发现**：支持mDNS本地节点发现（可配置启用/禁用）
- **安全传输**：基于libp2p的安全传输协议
- **可扩展性**：模块化设计，易于集成到不同类型的项目中

## 安装

```bash
go get github.com/lengzhao/network
```

## 快速开始

### 1. 创建网络实例

```go
import (
    "github.com/lengzhao/network"
)

// 方法1: 使用NewNetworkConfig创建默认配置（推荐）
cfg := network.NewNetworkConfig()
cfg.Host = "0.0.0.0"
cfg.Port = 8000
cfg.MaxPeers = 100
cfg.PrivateKeyPath = "./private_key.pem"
cfg.BootstrapPeers = []string{} // 可选的引导节点

// 创建网络实例
net, err := network.New(cfg)
if err != nil {
    log.Fatalf("Failed to create network: %v", err)
}
```

### 2. 注册消息处理器

```go
// 注册广播消息处理器
net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
    fmt.Printf("Received broadcast message from %s: %s\n", from, string(msg.Data))
    return nil
})

// 注册点对点请求处理器
net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
    // 回显请求数据作为响应
    return req.Data, nil
})
```

### 3. 注册消息过滤器（可选）

```go
// 注册消息过滤器
net.RegisterMessageFilter("chat", func(msg network.NetMessage) bool {
    // 过滤掉包含"spam"的消息
    if string(msg.Data) == "spam" {
        return false
    }
    return true
})
```

### 4. 启动网络

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 在goroutine中运行网络
go func() {
    if err := net.Run(ctx); err != nil {
        log.Printf("Network stopped with error: %v", err)
    }
}()

// 等待网络启动
time.Sleep(1 * time.Second)
```

### 5. 使用网络功能

```go
// 广播消息
err = net.BroadcastMessage("chat", []byte("Hello, world!"))
if err != nil {
    log.Printf("Failed to broadcast message: %v", err)
}

// 发送点对点请求
response, err := net.SendRequest(targetPeerID, "echo", []byte("Hello!"))
if err != nil {
    log.Printf("Failed to send request: %v", err)
} else {
    fmt.Printf("Received response: %s\n", string(response))
}
```

## 配置选项

### NetworkConfig 配置说明

| 字段名 | 类型 | 说明 | 默认值 |
|-------|------|------|--------|
| Host | string | 监听地址 | "0.0.0.0" |
| Port | int | 监听端口 | 0 |
| MaxPeers | int | 最大连接数 | 100 |
| PrivateKeyPath | string | 私钥文件路径 | "" |
| BootstrapPeers | []string | Bootstrap节点地址列表 | 空数组 |
| PeerWhitelist | []string | 节点白名单(节点ID字符串列表) | 空数组 |
| LogConfig | *log.LogConfig | 日志配置 | nil |
| DisableMDNS | bool | 是否禁用MDNS发现功能 | false (使用NewNetworkConfig()创建时) |
| EnablePeerScoring | bool | 是否启用Peer评分 | true |
| DiscoveryInterval | time.Duration | 节点发现间隔 | 1分钟 |
| MaxIPColocation | int | 单个IP地址最大节点数 | 3 |
| ScoreInspectInterval | time.Duration | 评分检查间隔 | 1分钟 |
| IPColocationWeight | float64 | IP共置权重 | -0.1 |
| BehaviourWeight | float64 | 行为权重 | -1.0 |
| BehaviourDecay | float64 | 行为衰减因子 | 0.98 |

**注意**: 当直接创建NetworkConfig结构体时，DisableMDNS的零值为false，表示默认启用MDNS。建议使用network.NewNetworkConfig()函数创建配置实例以获得正确的默认值。

## API参考

### NetworkInterface 接口

```go
type NetworkInterface interface {
    // Run 启动网络模块并运行
    Run(ctx context.Context) error

    // BroadcastMessage 广播消息到指定主题
    BroadcastMessage(topic string, data []byte) error

    // RegisterMessageHandler 注册广播消息处理器
    RegisterMessageHandler(topic string, handler MessageHandler)

    // RegisterRequestHandler 注册点对点请求处理器
    RegisterRequestHandler(requestType string, handler RequestHandler)

    // SendRequest 发送点对点请求
    SendRequest(peerID string, requestType string, data []byte) ([]byte, error)

    // ConnectToPeer 连接到指定节点
    ConnectToPeer(addr string) error

    // GetPeers 获取连接的节点列表
    GetPeers() []string

    // GetLocalAddresses 获取本地节点的地址列表
    GetLocalAddresses() []string

    // GetLocalPeerID 获取本地节点的Peer ID
    GetLocalPeerID() string

    // RegisterMessageFilter 注册广播消息过滤器
    RegisterMessageFilter(topic string, filter MessageFilter)
}
```

## 运行示例

```bash
# 运行基本示例
go run examples/base/basic_example.go

# 运行双节点示例
go run examples/two_node/two_node_example.go

# 运行MDNS配置示例
go run examples/mdns/mdns_example.go
```

## 测试

```bash
# 运行单元测试
go test -v ./network/...
```

## 依赖

- [go-libp2p](https://github.com/libp2p/go-libp2p) - libp2p Go实现
- [go-libp2p-pubsub](https://github.com/libp2p/go-libp2p-pubsub) - libp2p发布/订阅系统
- [go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) - Kademlia DHT实现

## 许可证

MIT