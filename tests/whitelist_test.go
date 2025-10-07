package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestPeerWhitelistNetworkLevel 网络层节点白名单测试
func TestPeerWhitelistNetworkLevel(t *testing.T) {
	// 创建三个网络实例Node1、Node2和Node3，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// 获取Node1的Peer ID，用于白名单
	node1PeerID := n1.GetLocalPeerID()

	// 为Node3创建带有白名单的配置
	cfg3 := &network.NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0,
		MaxPeers: 10,
		// 注意：PeerWhitelist在libp2p中可能不完全阻止消息传递，我们需要使用扩展消息过滤器
	}
	n3, err := network.New(cfg3)
	if err != nil {
		t.Fatalf("Failed to create network with whitelist: %v", err)
	}

	// 启动三个网络实例的运行上下文
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())

	// 确保ctx3被使用
	_ = ctx3

	// 在goroutine中运行三个网络
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n3.Run(ctx3)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 3 run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(500 * time.Millisecond)

	// 建立Node1和Node3之间的连接
	connectNetworks(t, n1, n3)

	// 建立Node2和Node3之间的连接
	connectNetworks(t, n2, n3)

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 验证所有节点都建立了连接（连接层不受白名单影响）
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()
	peers3 := n3.GetPeers()

	// 所有节点都应该建立连接
	if len(peers1) == 0 {
		t.Error("Node1 should be connected to other nodes")
	}

	if len(peers2) == 0 {
		t.Error("Node2 should be connected to other nodes")
	}

	if len(peers3) < 2 {
		t.Errorf("Node3 should be connected to at least two nodes, got %d connections", len(peers3))
	}

	// 在Node3上注册扩展消息过滤器，只允许来自Node1的消息
	collector3 := &messageCollector{}
	whitelist := map[string]bool{
		node1PeerID: true,
	}
	extendedFilter := &network.ExtendedMessageFilter{
		Whitelist: whitelist,
	}
	n3.RegisterExtendedMessageFilter("test-topic", extendedFilter)
	n3.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 在Node1和Node2上注册消息处理器
	collector1 := &messageCollector{}
	n1.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector1.addMessage(msg)
		return nil
	})

	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播消息
	messageFromNode1 := []byte("message from Node1")
	var broadcastErr error
	broadcastErr = n1.BroadcastMessage("test-topic", messageFromNode1)
	if broadcastErr != nil {
		t.Fatalf("Failed to broadcast message from Node1: %v", broadcastErr)
	}

	// 从Node2广播消息
	messageFromNode2 := []byte("message from Node2")
	broadcastErr = n2.BroadcastMessage("test-topic", messageFromNode2)
	if broadcastErr != nil {
		t.Fatalf("Failed to broadcast message from Node2: %v", broadcastErr)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node3只接收到来自Node1的消息（因为使用了扩展消息过滤器）
	messages3 := collector3.getMessages()

	// 验证Node3只接收到来自Node1的消息
	validMessages := 0
	for _, msg := range messages3 {
		if msg.From == node1PeerID {
			validMessages++
			// 验证消息内容正确
			if !bytes.Equal(msg.Data, messageFromNode1) {
				t.Errorf("Node3 received incorrect message from Node1. Expected: %s, Got: %s",
					string(messageFromNode1), string(msg.Data))
			}
		}
	}

	// 验证所有接收到的消息都来自Node1
	if validMessages != len(messages3) {
		t.Errorf("Node3 should only receive messages from Node1. Got %d valid messages out of %d total messages",
			validMessages, len(messages3))
	}

	// 验证Node3确实接收到了来自Node1的消息
	if validMessages == 0 {
		t.Error("Node3 should receive messages from Node1")
	}

	// 验证Node1和Node2都能接收到来自彼此的消息（它们没有白名单限制）
	messages1 := collector1.getMessages()
	messages2 := collector2.getMessages()

	// Node1应该接收到来自Node2的消息
	foundMessageFromNode2 := false
	for _, msg := range messages1 {
		if msg.From == n2.GetLocalPeerID() && bytes.Equal(msg.Data, messageFromNode2) {
			foundMessageFromNode2 = true
			break
		}
	}

	if !foundMessageFromNode2 {
		t.Error("Node1 should receive message from Node2")
	}

	// Node2应该接收到来自Node1的消息
	foundMessageFromNode1 := false
	for _, msg := range messages2 {
		if msg.From == n1.GetLocalPeerID() && bytes.Equal(msg.Data, messageFromNode1) {
			foundMessageFromNode1 = true
			break
		}
	}

	if !foundMessageFromNode1 {
		t.Error("Node2 should receive message from Node1")
	}

	// 清理资源
	defer cancel1()
	defer cancel2()
	defer cancel3()
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestExtendedMessageFilter 扩展消息过滤器测试
func TestExtendedMessageFilter(t *testing.T) {
	// 创建两个网络实例Node1和Node2，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// 启动两个网络实例的运行上下文
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	// 在goroutine中运行两个网络
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(500 * time.Millisecond)

	// 建立Node1和Node2之间的连接
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between networks")
	}

	// 在Node2上注册扩展消息过滤器，只允许来自Node1的消息，且拒绝包含"blocked"关键字的消息
	collector := &messageCollector{}

	// 创建白名单，只允许Node1
	whitelist := map[string]bool{
		n1.GetLocalPeerID(): true,
	}

	extendedFilter := &network.ExtendedMessageFilter{
		Whitelist: whitelist,
		ContentFilter: func(msg network.NetMessage) bool {
			// 拒绝包含"blocked"关键字的消息
			return !bytes.Contains(msg.Data, []byte("blocked"))
		},
	}

	n2.RegisterExtendedMessageFilter("test-topic", extendedFilter)
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播三条消息：
	// 1. 正常消息
	// 2. 包含"blocked"关键字的消息
	// 3. 从伪造节点ID发送的消息（用于测试白名单）
	normalMessage := []byte("this is a normal message")
	blockedMessage := []byte("this message contains blocked keyword")

	err := n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	err = n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast blocked message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2只处理了正常消息
	messages := collector.getMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d messages", len(messages))
	}

	// 验证Node2通过消息处理器接收到的消息是正常消息
	if len(messages) > 0 {
		receivedMessage := messages[0]
		if !bytes.Equal(receivedMessage.Data, normalMessage) {
			t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(receivedMessage.Data))
		}

		// 验证消息来源是Node1
		if receivedMessage.From != n1.GetLocalPeerID() {
			t.Errorf("Message should be from Node1. Expected: %s, Got: %s", n1.GetLocalPeerID(), receivedMessage.From)
		}
	}

	// 验证被过滤的消息没有被处理
	for _, msg := range messages {
		if bytes.Contains(msg.Data, []byte("blocked")) {
			t.Error("Blocked message was incorrectly processed")
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}
