package network

import (
	"context"
	"fmt"

	"github.com/lengzhao/network/log"
)

// NetMessage 广播模式下的消息结构
type NetMessage struct {
	From  string // 消息发送方的Peer ID
	Topic string // 消息所属主题
	Data  []byte // 消息数据
}

// Request 点对点请求结构
type Request struct {
	Type string // 请求类型
	Data []byte // 请求数据
}

// Serialize 序列化请求
func (r *Request) Serialize() ([]byte, error) {
	// 简单的序列化实现，实际项目中可以使用更复杂的序列化方式如protobuf
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize 反序列化请求
func (r *Request) Deserialize(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	typeLen := int(data[0])
	if len(data) < 1+typeLen {
		return fmt.Errorf("invalid data format")
	}
	r.Type = string(data[1 : 1+typeLen])
	r.Data = data[1+typeLen:]
	return nil
}

// Response 点对点响应结构
type Response struct {
	Type string // 响应类型
	Data []byte // 响应数据
}

// Serialize 序列化响应
func (r *Response) Serialize() ([]byte, error) {
	// 简单的序列化实现，实际项目中可以使用更复杂的序列化方式如protobuf
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize 反序列化响应
func (r *Response) Deserialize(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	typeLen := int(data[0])
	if len(data) < 1+typeLen {
		return fmt.Errorf("invalid data format")
	}
	r.Type = string(data[1 : 1+typeLen])
	r.Data = data[1+typeLen:]
	return nil
}

// MessageHandler 用于处理广播模式下接收到的消息
type MessageHandler func(from string, msg NetMessage) error

// RequestHandler 用于处理点对点模式下的请求
type RequestHandler func(from string, req Request) ([]byte, error)

// MessageFilter 用于过滤广播消息
type MessageFilter func(msg NetMessage) bool

// ExtendedMessageFilter 扩展的消息过滤器，支持节点白名单和内容过滤
type ExtendedMessageFilter struct {
	Whitelist     map[string]bool // 节点白名单
	ContentFilter MessageFilter   // 内容过滤器
}

// Filter 应用过滤规则
func (f *ExtendedMessageFilter) Filter(msg NetMessage) bool {
	// 检查发送方是否在白名单中（如果配置了白名单）
	if len(f.Whitelist) > 0 {
		if !f.Whitelist[msg.From] {
			return false
		}
	}

	// 应用内容过滤器
	if f.ContentFilter != nil {
		return f.ContentFilter(msg)
	}

	return true
}

// NetworkInterface 网络接口，定义了所有对外提供的功能
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

	// RegisterExtendedMessageFilter 注册扩展消息过滤器
	RegisterExtendedMessageFilter(topic string, filter *ExtendedMessageFilter)

	// GetLogManager 获取日志管理器
	GetLogManager() log.LogManager
}
