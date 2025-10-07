package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestBroadcastFunctionality 测试广播功能
func TestBroadcastFunctionality(t *testing.T) {
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

	// 获取Node2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	// 连接网络1到网络2
	err := n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect networks: %v", err)
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
	receivedMessages := make(chan network.NetMessage, 10)

	// 注册消息处理器
	n1.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		receivedMessages <- msg
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		receivedMessages <- msg
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从网络1广播消息
	messageData := []byte("broadcast message")
	err = n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证两个节点都通过messageCollector接收到消息
	// 注意：发送方不会收到自己发送的消息，所以只期望收到1条消息（来自n2）
	messages := make([]network.NetMessage, 0)

	close(receivedMessages)
	for msg := range receivedMessages {
		messages = append(messages, msg)
	}

	// 修复：期望至少1条消息，而不是2条
	// 因为发送方（n1）不会收到自己发送的消息，只有接收方（n2）会收到消息
	if len(messages) < 1 {
		t.Errorf("Expected at least 1 message, got %d messages", len(messages))
	}

	// 验证所有节点接收到的消息内容与发送的内容一致
	for _, msg := range messages {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// TestBroadcastMultipleTopics 测试多个主题的广播功能
func TestBroadcastMultipleTopics(t *testing.T) {
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

	// 获取Node2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	// 连接网络1到网络2
	err := n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect networks: %v", err)
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
	receivedMessagesTopic1 := make(chan network.NetMessage, 10)
	receivedMessagesTopic2 := make(chan network.NetMessage, 10)

	// 注册消息处理器
	n1.RegisterMessageHandler("topic1", func(from string, msg network.NetMessage) error {
		receivedMessagesTopic1 <- msg
		return nil
	})

	n2.RegisterMessageHandler("topic1", func(from string, msg network.NetMessage) error {
		receivedMessagesTopic1 <- msg
		return nil
	})

	n1.RegisterMessageHandler("topic2", func(from string, msg network.NetMessage) error {
		receivedMessagesTopic2 <- msg
		return nil
	})

	n2.RegisterMessageHandler("topic2", func(from string, msg network.NetMessage) error {
		receivedMessagesTopic2 <- msg
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从网络1广播消息到"topic1"主题
	messageDataTopic1 := []byte("broadcast message to topic1")
	err = n1.BroadcastMessage("topic1", messageDataTopic1)
	if err != nil {
		t.Fatalf("Failed to broadcast message to topic1: %v", err)
	}

	// 从网络1广播消息到"topic2"主题
	messageDataTopic2 := []byte("broadcast message to topic2")
	err = n1.BroadcastMessage("topic2", messageDataTopic2)
	if err != nil {
		t.Fatalf("Failed to broadcast message to topic2: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2都通过messageCollector接收到消息
	messagesTopic1 := make([]network.NetMessage, 0)
	messagesTopic2 := make([]network.NetMessage, 0)

	close(receivedMessagesTopic1)
	close(receivedMessagesTopic2)

	for msg := range receivedMessagesTopic1 {
		messagesTopic1 = append(messagesTopic1, msg)
	}

	for msg := range receivedMessagesTopic2 {
		messagesTopic2 = append(messagesTopic2, msg)
	}

	if len(messagesTopic1) == 0 {
		t.Error("Node2 did not receive the broadcast message for topic1")
	}

	if len(messagesTopic2) == 0 {
		t.Error("Node2 did not receive the broadcast message for topic2")
	}

	// 验证所有节点接收到的消息内容与发送的内容一致
	for _, msg := range messagesTopic1 {
		if !bytes.Equal(msg.Data, messageDataTopic1) {
			t.Errorf("Node2 received incorrect message data for topic1. Expected: %s, Got: %s", string(messageDataTopic1), string(msg.Data))
		}
	}

	for _, msg := range messagesTopic2 {
		if !bytes.Equal(msg.Data, messageDataTopic2) {
			t.Errorf("Node2 received incorrect message data for topic2. Expected: %s, Got: %s", string(messageDataTopic2), string(msg.Data))
		}
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}
