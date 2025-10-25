package tests

import (
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestPeerScoringBasic 测试基本的Peer评分功能
func TestPeerScoringBasic(t *testing.T) {
	// 创建两个网络实例
	cfg1 := &network.NetworkConfig{
		Host:              "127.0.0.1",
		Port:              0,
		MaxPeers:          10,
		EnablePeerScoring: true, // 启用Peer评分
	}

	cfg2 := &network.NetworkConfig{
		Host:              "127.0.0.1",
		Port:              0,
		MaxPeers:          10,
		EnablePeerScoring: true, // 启用Peer评分
	}

	n1 := createTestNetworkWithConfig(t, cfg1)
	n2 := createTestNetworkWithConfig(t, cfg2)

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

	// 获取Node2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	err := n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect to peer: %v", err)
	}

	// 等待连接建立
	time.Sleep(500 * time.Millisecond)

	// 验证连接状态
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	if len(peers1) == 0 {
		t.Error("Network 1 has no peers")
	}

	if len(peers2) == 0 {
		t.Error("Network 2 has no peers")
	}

	// 等待一段时间让评分检查器被调用
	time.Sleep(3 * time.Second)

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// TestPeerScoringWithBroadcast 测试Peer评分与广播消息的结合
func TestPeerScoringWithBroadcast(t *testing.T) {
	// 创建两个网络实例
	cfg1 := &network.NetworkConfig{
		Host:              "127.0.0.1",
		Port:              0,
		MaxPeers:          10,
		EnablePeerScoring: true,
	}

	cfg2 := &network.NetworkConfig{
		Host:              "127.0.0.1",
		Port:              0,
		MaxPeers:          10,
		EnablePeerScoring: true,
	}

	n1 := createTestNetworkWithConfig(t, cfg1)
	n2 := createTestNetworkWithConfig(t, cfg2)

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

	// 获取Node2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	err := n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect to peer: %v", err)
	}

	// 等待连接建立
	time.Sleep(500 * time.Millisecond)

	// 验证连接状态
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	if len(peers1) == 0 {
		t.Error("Network 1 has no peers")
	}

	if len(peers2) == 0 {
		t.Error("Network 2 has no peers")
	}

	// 用于接收消息的通道
	receivedMessages := make(chan NetMessage, 10)

	// 注册消息处理器
	n1.RegisterMessageHandler("test-topic", func(from string, topic string, data []byte) error {
		receivedMessages <- NetMessage{From: from, Topic: topic, Data: data}
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, topic string, data []byte) error {
		receivedMessages <- NetMessage{From: from, Topic: topic, Data: data}
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从网络1广播消息
	messageData := []byte("broadcast message for scoring test")
	err = n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证消息被接收
	messages := make([]NetMessage, 0)
	close(receivedMessages)
	for msg := range receivedMessages {
		messages = append(messages, msg)
	}

	if len(messages) < 1 {
		t.Errorf("Expected at least 1 message, got %d messages", len(messages))
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// createTestNetworkWithConfig 创建带配置的测试网络实例
func createTestNetworkWithConfig(t *testing.T, cfg *network.NetworkConfig) network.NetworkInterface {
	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}
	return n
}
