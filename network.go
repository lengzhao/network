package network

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

// PeerScoreInspector Peer评分检查器接口
// type PeerScoreInspector func(map[peer.ID]*pubsub.PeerScoreSnapshot) // 已移至types.go

// Network 网络层 - 基于libp2p的现代化P2P网络
type Network struct {
	config *NetworkConfig
	host   host.Host
	ctx    context.Context
	cancel context.CancelFunc

	// DHT自动管理（由libp2p.Routing自动处理）

	// Gossipsub消息传播
	pubsub   *pubsub.PubSub
	topics   map[string]*pubsub.Topic
	topicsMu sync.RWMutex

	// 消息处理
	messageHandlers map[string]MessageHandler
	handlersMu      sync.RWMutex

	// 消息过滤
	messageFilters map[string]MessageFilter
	filtersMu      sync.RWMutex

	// 点对点请求处理
	requestHandlers map[string]RequestHandler
	requestMu       sync.RWMutex

	// 节点发现
	mdnsService mdns.Service
}

// topicValidatorHandler 主题验证器处理函数
func (n *Network) topicValidatorHandler(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	topicName := msg.GetTopic()

	// 检查是否有注册的MessageFilter
	n.filtersMu.RLock()
	filter, exists := n.messageFilters[topicName]
	n.filtersMu.RUnlock()

	// 如果有MessageFilter，则应用过滤
	if exists {
		netMsg := NetMessage{
			From:  msg.ReceivedFrom.String(),
			Topic: topicName,
			Data:  msg.Data,
		}

		// 如果过滤器返回false，则拒绝消息
		if !filter(netMsg) {
			return pubsub.ValidationReject // 拒绝消息
		}
	}

	// 默认接受消息
	return pubsub.ValidationAccept
}

// autoRegisterTopicValidator 在joinPubsub时自动注册主题验证器
func (n *Network) autoRegisterTopicValidator(topic string) error {
	return n.pubsub.RegisterTopicValidator(topic, n.topicValidatorHandler)
}

var _ NetworkInterface = (*Network)(nil)

// New 创建新的网络实例
func New(cfg *NetworkConfig, ops ...libp2p.Option) (NetworkInterface, error) {
	// 如果配置为nil，使用默认配置
	if cfg == nil {
		cfg = NewNetworkConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 获取私钥
	priv, err := getOrGeneratePrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to get private key", "error", err)
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// 创建连接管理器
	connManager, err := connmgr.NewConnManager(
		20,                                     // 最小连接数（默认值）
		cfg.MaxPeers,                           // 最大连接数
		connmgr.WithGracePeriod(time.Minute*5), // 优雅期
	)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to create connection manager", "error", err)
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// 解析 bootstrap 节点
	var bootstrapPeers []peer.AddrInfo
	for _, peerAddr := range cfg.BootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			slog.With("module", "network").Warn("failed to parse bootstrap peer address", "address", peerAddr, "error", err)
			continue
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			slog.With("module", "network").Warn("failed to parse bootstrap peer info", "address", peerAddr, "error", err)
			continue
		}

		bootstrapPeers = append(bootstrapPeers, *addrInfo)
	}
	// if len(bootstrapPeers) == 0 {
	// 	bootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()
	// }
	options := []libp2p.Option{
		// 使用生成的密钥对
		libp2p.Identity(priv),
		// 监听地址
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Host, cfg.Port),         // TCP连接
			fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", cfg.Host, cfg.Port), // QUIC传输
		),
		// 支持noise连接
		libp2p.Security(noise.ID, noise.New),
		// 支持默认传输协议
		libp2p.DefaultTransports,
		// 尝试使用uPNP为NAT主机开放端口
		libp2p.NATPortMap(),
		// 启用NAT服务以帮助其他节点检测NAT
		libp2p.EnableNATService(),
		// 启用自动中继
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{}),
		// 启用洞穿
		libp2p.EnableHolePunching(),
		// 使用连接管理器
		libp2p.ConnectionManager(connManager),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dhtInstance, err := dht.New(context.Background(), h, dht.BootstrapPeers(bootstrapPeers...))
			return dhtInstance, err
		}),
	}
	options = append(options, ops...)

	// 创建libp2p主机
	host, err := libp2p.New(
		options...,
	)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to create libp2p host", "error", err)
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	n := &Network{
		config:          cfg,
		host:            host,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string]MessageHandler),
		messageFilters:  make(map[string]MessageFilter),
		requestHandlers: make(map[string]RequestHandler),
		topics:          make(map[string]*pubsub.Topic),
	}

	// DHT由libp2p.Routing自动初始化

	// 初始化Gossipsub
	if err := n.initializeGossipsub(); err != nil {
		cancel()
		slog.With("module", "network").Error("failed to initialize Gossipsub", "error", err)
		return nil, fmt.Errorf("failed to initialize Gossipsub: %w", err)
	}

	n.host.SetStreamHandler("/network/1.0.0/request", n.handleRequest)

	// 设置网络通知
	host.Network().Notify(n)

	return n, nil
}

// Run 运行网络
func (n *Network) Run(ctx context.Context) error {
	n.startMDNSDiscovery()

	slog.With("module", "network").Info("network started", "addresses", n.GetLocalAddresses())

	// 等待context取消
	<-ctx.Done()

	// 停止mDNS服务（如果已启动）
	if n.mdnsService != nil {
		n.mdnsService.Close()
	}

	// 关闭主机
	if n.host != nil {
		n.host.Close()
	}

	return ctx.Err()
}

// initializeGossipsub 初始化Gossipsub
func (n *Network) initializeGossipsub() error {
	var opts []pubsub.Option

	// 启用消息签名以确保消息来源可验证
	opts = append(opts, pubsub.WithMessageSignaturePolicy(pubsub.StrictSign))
	opts = append(opts, pubsub.WithMessageIdFn(func(pmsg *pubsub_pb.Message) string {
		// 使用消息数据和发送者ID生成唯一的消息ID
		h := sha256.Sum256(pmsg.Data)
		return string(h[:])
	}))

	// 如果启用了Peer评分，则配置Peer评分系统
	if n.config.EnablePeerScoring {
		// 设置Peer评分参数来控制发布权限
		scoreParams := &pubsub.PeerScoreParams{
			Topics: map[string]*pubsub.TopicScoreParams{},
			AppSpecificScore: func(p peer.ID) float64 {
				return n.config.AppSpecificScore
			},
			AppSpecificWeight:           1.0,
			IPColocationFactorWeight:    n.config.IPColocationWeight,
			IPColocationFactorThreshold: n.config.MaxIPColocation,
			BehaviourPenaltyWeight:      n.config.BehaviourWeight,
			BehaviourPenaltyDecay:       n.config.BehaviourDecay,
			DecayInterval:               time.Hour,
			DecayToZero:                 0.01,
			RetainScore:                 24 * time.Hour,
		}

		// 设置Peer评分阈值
		thresholds := &pubsub.PeerScoreThresholds{
			GossipThreshold:             -100, // 低于此阈值时抑制gossip传播
			PublishThreshold:            -200, // 低于此阈值时不应发布消息（必须 <= GossipThreshold）
			GraylistThreshold:           -500, // 低于此阈值时完全抑制消息处理（必须 <= PublishThreshold）
			AcceptPXThreshold:           1000, // 高于此阈值时接受PX
			OpportunisticGraftThreshold: 50,   // 触发机会性graft的网格评分中位数阈值
		}

		opts = append(opts, pubsub.WithPeerScore(scoreParams, thresholds))

	}

	// 创建Gossipsub实例
	pubsubInstance, err := pubsub.NewGossipSub(n.ctx, n.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Gossipsub: %w", err)
	}

	n.pubsub = pubsubInstance
	return nil
}

// GetPeers 获取连接的节点
func (n *Network) GetPeers() []string {
	peers := n.host.Network().Peers()
	result := make([]string, len(peers))
	for i, p := range peers {
		result[i] = p.String()
	}

	return result
}

// ConnectToPeer 连接到指定节点（外部API）
func (n *Network) ConnectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	if err := n.host.Connect(n.ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// GetLocalAddresses 获取本地节点的地址列表
func (n *Network) GetLocalAddresses() []string {
	if n.host == nil {
		return []string{}
	}

	addresses := n.host.Addrs()
	result := make([]string, len(addresses))

	for i, addr := range addresses {
		// 将地址和节点ID组合成完整的p2p地址
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), n.host.ID().String())
		result[i] = fullAddr
	}

	return result
}

// BroadcastMessage 广播消息
func (n *Network) BroadcastMessage(topic string, data []byte) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topicObj, err := n.getOrCreateTopic(topic)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	if err := topicObj.Publish(n.ctx, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// getOrCreateTopic 获取或创建主题
func (n *Network) getOrCreateTopic(topicName string) (*pubsub.Topic, error) {
	n.topicsMu.Lock()
	defer n.topicsMu.Unlock()

	if topic, exists := n.topics[topicName]; exists {
		return topic, nil
	}

	topic, err := n.pubsub.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	// 自动注册主题验证器
	if err := n.autoRegisterTopicValidator(topicName); err != nil {
		// 记录日志但不中断流程
		slog.With("module", "network.pubsub").Warn("failed to register topic validator", "topic", topicName, "error", err)
	}

	n.topics[topicName] = topic
	return topic, nil
}

// subscribeToTopicInternal 内部订阅主题方法
func (n *Network) subscribeToTopicInternal(topicName string) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topic, err := n.getOrCreateTopic(topicName)
	if err != nil {
		return fmt.Errorf("获取主题失败: %w", err)
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	go n.handleSubscription(topicName, subscription)
	return nil
}

// handleSubscription 处理订阅
func (n *Network) handleSubscription(topicName string, subscription *pubsub.Subscription) {
	defer subscription.Cancel()

	for {
		msg, err := subscription.Next(n.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			slog.With("module", "network.pubsub").Error("failed to receive message", "error", err)
			continue
		}

		if msg.ReceivedFrom == n.host.ID() {
			continue // 忽略自己发送的消息
		}

		go n.processPubsubMessage(topicName, msg)
	}
}

// processPubsubMessage 处理pubsub消息
func (n *Network) processPubsubMessage(topicName string, msg *pubsub.Message) {
	// 应用过滤器
	n.filtersMu.RLock()

	// 检查是否有普通过滤器
	filter, filterExists := n.messageFilters[topicName]

	n.filtersMu.RUnlock()

	// 应用普通过滤器
	if filterExists {
		netMsg := NetMessage{
			From:  msg.ReceivedFrom.String(),
			Topic: topicName,
			Data:  msg.Data,
		}

		// 如果过滤器返回false，则丢弃消息
		if !filter(netMsg) {
			return
		}
	}

	// 调用消息处理器
	n.handlersMu.RLock()
	handler, exists := n.messageHandlers[topicName]
	n.handlersMu.RUnlock()

	if !exists {
		return
	}

	netMsg := NetMessage{
		From:  msg.ReceivedFrom.String(),
		Topic: topicName,
		Data:  msg.Data,
	}

	if err := handler(msg.ReceivedFrom.String(), netMsg); err != nil {
		slog.With("module", "network.pubsub").Error("failed to process message", "error", err)
	}
}

// RegisterMessageHandler 注册消息处理器
// 自动订阅相应的主题以接收消息
func (n *Network) RegisterMessageHandler(topic string, handler MessageHandler) {
	n.handlersMu.Lock()
	defer n.handlersMu.Unlock()

	// 检查是否已经注册过该主题的处理器
	_, exists := n.messageHandlers[topic]
	n.messageHandlers[topic] = handler

	// 如果是新注册的主题，自动订阅
	if !exists {
		go func() {
			if err := n.subscribeToTopicInternal(topic); err != nil {
				slog.With("module", "network.pubsub").Warn("failed to auto-subscribe to topic", "topic", topic, "error", err)
			} else {
				slog.With("module", "network.pubsub").Info("auto-subscribed to topic", "topic", topic)
			}
		}()
	}
}

// RegisterMessageFilter 注册消息过滤器
func (n *Network) RegisterMessageFilter(topic string, filter MessageFilter) {
	n.filtersMu.Lock()
	defer n.filtersMu.Unlock()
	n.messageFilters[topic] = filter
}

// handleRequest 处理点对点请求
func (n *Network) handleRequest(stream libp2pnetwork.Stream) {
	// 使用defer确保流在函数退出时被正确关闭
	defer stream.Close()

	slog.With("module", "network.request").Info("handling request from peer", "peer", stream.Conn().RemotePeer().String())

	// 设置读取超时
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		slog.With("module", "network.request").Error("failed to set read deadline", "error", err)
		return
	}

	// 使用更高效的IO读取方式
	data, err := io.ReadAll(stream)
	if err != nil {
		slog.With("module", "network.request").Error("failed to read request data", "error", err)
		return
	}

	// 重置超时设置，准备写入响应
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		slog.With("module", "network.request").Error("failed to set write deadline", "error", err)
		return
	}

	if len(data) == 0 {
		slog.With("module", "network.request").Warn("received empty request data")
		return
	}

	// 解析请求
	var req Request
	if err := req.Deserialize(data); err != nil {
		slog.With("module", "network.request").Error("failed to deserialize request", "error", err)
		// 发送错误响应
		resp := Response{
			Type: "error",
			Data: []byte("failed to deserialize request: " + err.Error()),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	slog.With("module", "network.request").Info("parsed request", "type", req.Type, "data_length", len(req.Data))

	// 获取请求处理器
	n.requestMu.RLock()
	handler, exists := n.requestHandlers[req.Type]
	n.requestMu.RUnlock()

	if !exists {
		slog.With("module", "network.request").Warn("no handler found for request type", "type", req.Type)
		// 发送错误响应
		resp := Response{
			Type: "error",
			Data: []byte("no handler found for request type: " + req.Type),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	// 处理请求
	respData, err := handler(stream.Conn().RemotePeer().String(), req)
	if err != nil {
		slog.With("module", "network.request").Error("failed to process request", "error", err)
		// 发送错误响应
		resp := Response{
			Type: "error",
			Data: []byte(err.Error()),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	// 发送响应
	resp := Response{
		Type: req.Type,
		Data: respData,
	}
	respBytes, err := resp.Serialize()
	if err != nil {
		slog.With("module", "network.request").Error("failed to serialize response", "error", err)
		// 发送序列化错误响应
		errorResp := Response{
			Type: "error",
			Data: []byte("failed to serialize response: " + err.Error()),
		}
		errorRespBytes, _ := errorResp.Serialize()
		if _, err := stream.Write(errorRespBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send serialization error response", "error", err)
		}
		return
	}

	slog.With("module", "network.request").Info("sending response", "type", req.Type, "response_bytes", len(respBytes))
	if _, err := stream.Write(respBytes); err != nil {
		slog.With("module", "network.request").Error("failed to send response", "error", err)
	} else {
		slog.With("module", "network.request").Info("response sent successfully")
		stream.CloseWrite()
	}
}

// RegisterRequestHandler 注册点对点请求处理器
func (n *Network) RegisterRequestHandler(requestType string, handler RequestHandler) {
	n.requestMu.Lock()
	defer n.requestMu.Unlock()
	n.requestHandlers[requestType] = handler
	slog.With("module", "network.request").Info("registered request handler for type", "type", requestType)
}

// SendRequest 发送点对点请求
func (n *Network) SendRequest(peerID string, requestType string, data []byte) ([]byte, error) {
	pID, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}
	ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
	defer cancel()

	// 创建流
	stream, err := n.host.NewStream(ctx, pID, "/network/1.0.0/request")
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// 设置写入超时
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// 构造请求
	req := Request{
		Type: requestType,
		Data: data,
	}

	// 序列化请求
	reqData, err := req.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	// 发送请求
	if _, err := stream.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 确保请求数据被发送
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("failed to close write: %w", err)
	}

	// 设置读取超时
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// 使用更高效的IO读取方式
	responseData, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.With("module", "network.request").Info("read response", "bytes_read", len(responseData))

	// 检查是否有响应数据
	if len(responseData) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	// 解析响应
	var resp Response
	if err := resp.Deserialize(responseData); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	// 检查是否是错误响应
	if resp.Type == "error" {
		return nil, fmt.Errorf("remote error: %s", string(resp.Data))
	}

	return resp.Data, nil
}

// getOrGeneratePrivateKey 获取或生成私钥
func getOrGeneratePrivateKey(keyPath string) (crypto.PrivKey, error) {
	// 如果指定了私钥文件路径
	if keyPath != "" {
		// 检查文件是否存在
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			// 文件不存在，生成新私钥并保存
			slog.With("module", "network.security").Info("private key file not found, generating new key", "path", keyPath)
			priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %w", err)
			}

			// 保存私钥到文件
			if err := SavePrivateKeyToFile(priv, keyPath); err != nil {
				return nil, fmt.Errorf("failed to save private key: %w", err)
			}

			slog.With("module", "network.security").Info("new private key saved", "path", keyPath)
			return priv, nil
		}

		// 文件存在，尝试读取
		priv, err := loadPrivateKeyFromFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		slog.With("module", "network.security").Info("private key loaded from file", "path", keyPath)
		return priv, nil
	}

	// 没有指定文件路径，生成临时私钥
	slog.With("module", "network.security").Info("no private key path specified, generating temporary key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return priv, nil
}

// loadPrivateKeyFromFile 从文件加载私钥
func loadPrivateKeyFromFile(path string) (crypto.PrivKey, error) {
	// 读取私钥文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// 解析PEM格式
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM format private key file")
	}

	// 解析私钥
	priv, err := crypto.UnmarshalPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return priv, nil
}

// SavePrivateKeyToFile 保存私钥到文件
func SavePrivateKeyToFile(priv crypto.PrivKey, path string) error {
	// 序列化私钥
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to serialize private key: %w", err)
	}

	// 创建PEM块
	block := &pem.Block{
		Type:  "LIBP2P PRIVATE KEY",
		Bytes: privBytes,
	}

	// 编码为PEM格式
	pemData := pem.EncodeToMemory(block)

	// 写入文件
	err = os.WriteFile(path, pemData, 0600) // 只有所有者可读写
	if err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	return nil
}

// startMDNSDiscovery 启动mDNS发现
func (n *Network) startMDNSDiscovery() {
	// 创建mDNS服务，服务名使用_network_discovery
	n.mdnsService = mdns.NewMdnsService(n.host, "_network_discovery", n)

	// 启动mDNS服务
	if err := n.mdnsService.Start(); err != nil {
		slog.With("module", "network.discovery").Error("failed to start mDNS service", "error", err)
		return
	}

	slog.With("module", "network.discovery").Info("mDNS service started with service name: _network_discovery")
}

// HandlePeerFound 实现mdns.Notifee接口，处理发现的节点
func (n *Network) HandlePeerFound(pi peer.AddrInfo) {
	// 避免连接自己
	if pi.ID == n.host.ID() {
		return
	}

	slog.With("module", "network.discovery").Info("mDNS peer discovered", "peer_id", pi.ID.String(), "addresses", pi.Addrs)

	if err := n.host.Connect(n.ctx, pi); err != nil {
		slog.With("module", "network.discovery").Warn("failed to connect to discovered peer", "peer_id", pi.ID.String(), "error", err)
	} else {
		slog.With("module", "network.discovery").Info("successfully connected to discovered peer", "peer_id", pi.ID.String())
	}
}

// 实现network.Notifiee接口
func (n *Network) Listen(libp2pnetwork.Network, multiaddr.Multiaddr)      {}
func (n *Network) ListenClose(libp2pnetwork.Network, multiaddr.Multiaddr) {}
func (n *Network) Connected(net libp2pnetwork.Network, conn libp2pnetwork.Conn) {
	peerID := conn.RemotePeer()
	slog.With("module", "network.connection").Info("peer connected", "peer_id", peerID.String())
}
func (n *Network) Disconnected(net libp2pnetwork.Network, conn libp2pnetwork.Conn) {
	peerID := conn.RemotePeer()
	slog.With("module", "network.connection").Info("peer disconnected", "peer_id", peerID.String())
}
func (n *Network) OpenedStream(libp2pnetwork.Network, libp2pnetwork.Stream) {}
func (n *Network) ClosedStream(libp2pnetwork.Network, libp2pnetwork.Stream) {}

// IsPeerConnected 检查节点是否已连接
func (n *Network) IsPeerConnected(peerID peer.ID) bool {
	conns := n.host.Network().ConnsToPeer(peerID)
	return len(conns) > 0
}

// GetLocalPeerID 获取本地节点的Peer ID
func (n *Network) GetLocalPeerID() string {
	if n.host == nil {
		return ""
	}
	return n.host.ID().String()
}
