package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestMessageFilterBasic 基本消息过滤测试
func TestMessageFilterBasic(t *testing.T) {
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

	// 在Node2上注册消息处理器和消息过滤器，过滤器拒绝包含"filtered"关键字的消息
	collector := &messageCollector{}

	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		// 拒绝包含"filtered"关键字的消息
		return !bytes.Contains(msg.Data, []byte("filtered"))
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播两条消息：一条包含"filtered"关键字，另一条不包含
	filteredMessage := []byte("this message should be filtered")
	normalMessage := []byte("this message should be received")

	err := n1.BroadcastMessage("test-topic", filteredMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast filtered message: %v", err)
	}

	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2只处理不包含"filtered"关键字的消息
	messages := collector.getMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d messages", len(messages))
	}

	// 验证Node2通过消息处理器接收到的消息数量正确
	if len(messages) > 0 {
		receivedMessage := messages[0]
		if !bytes.Equal(receivedMessage.Data, normalMessage) {
			t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(receivedMessage.Data))
		}
	}

	// 验证被过滤的消息没有被处理
	for _, msg := range messages {
		if bytes.Contains(msg.Data, []byte("filtered")) {
			t.Error("Filtered message was incorrectly processed")
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterNoForward 过滤后不转发测试
func TestMessageFilterNoForward(t *testing.T) {
	// 创建三个网络实例Node1、Node2和Node3，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)
	n3 := createTestNetwork(t, "127.0.0.1", 0)

	// 启动三个网络实例的运行上下文
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	defer cancel3()

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

	// 在Node2上注册拒绝包含"do-not-forward"关键字消息的过滤器
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		t.Logf("Filtered message: %t", !bytes.Contains(msg.Data, []byte("do-not-forward")))
		// 拒绝包含"do-not-forward"关键字的消息
		return !bytes.Contains(msg.Data, []byte("do-not-forward"))
	})

	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		t.Logf("Received message: %s", string(msg.Data))
		collector2.addMessage(msg)
		return nil
	})

	// 在Node3上注册消息处理器
	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		t.Logf("Received message: %s", string(msg.Data))
		collector3.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播包含"do-not-forward"关键字的消息
	filteredMessage := []byte("this message should not be forwarded: do-not-forward")
	err := n1.BroadcastMessage("test-topic", filteredMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast filtered message: %v", err)
	}

	// 等待足够时间后验证Node3没有接收到该消息
	time.Sleep(2 * time.Second)

	messages3 := collector3.getMessages()
	if len(messages3) > 0 {
		t.Error("Node3 received message that should have been filtered by Node2")
	}

	// 从Node1广播不包含"do-not-forward"关键字的消息
	normalMessage := []byte("this message should be forwarded normally")
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	messages2 := collector2.getMessages()
	if len(messages2) == 0 {
		t.Error("Node2 did not receive the normal message")
	}

	// 验证Node3成功接收到该消息
	messages3 = collector3.getMessages()
	if len(messages3) == 0 {
		t.Error("Node3 did not receive the normal message")
	}

	for _, msg := range messages3 {
		if !bytes.Equal(msg.Data, normalMessage) {
			t.Errorf("Node3 received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(msg.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestMessageFilterDynamic 动态过滤规则测试
func TestMessageFilterDynamic(t *testing.T) {
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

	// 在Node2上注册消息处理器和初始过滤器
	collector := &messageCollector{}

	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	// 初始过滤器：拒绝包含"blocked"关键字的消息
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("blocked"))
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播包含"blocked"关键字的消息
	blockedMessage := []byte("this message is blocked")
	err := n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast blocked message: %v", err)
	}

	// 从Node1广播正常消息
	normalMessage := []byte("this is a normal message")
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证初始过滤器生效：只接收到正常消息
	messages := collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message with initial filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, normalMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(messages[0].Data))
	}

	// 清空消息收集器
	collector.clear()

	// 更新过滤器：现在拒绝包含"forbidden"关键字的消息
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("forbidden"))
	})

	// 等待过滤器更新生效
	time.Sleep(500 * time.Millisecond)

	// 从Node1广播包含"forbidden"关键字的消息
	forbiddenMessage := []byte("this message is forbidden")
	err = n1.BroadcastMessage("test-topic", forbiddenMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast forbidden message: %v", err)
	}

	// 从Node1广播正常消息
	anotherNormalMessage := []byte("another normal message")
	err = n1.BroadcastMessage("test-topic", anotherNormalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast another normal message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证更新后的过滤器生效：只接收到正常消息
	messages = collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message with updated filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, anotherNormalMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(anotherNormalMessage), string(messages[0].Data))
	}

	// 验证之前被允许的消息现在被拒绝（blocked消息应该能通过新的过滤器）
	collector.clear()

	// 重新发送之前被blocked的消息
	err = n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to rebroadcast blocked message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证blocked消息现在能通过新的过滤器
	messages = collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message (previously blocked) with updated filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, blockedMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(blockedMessage), string(messages[0].Data))
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterMultiple 多过滤器协同测试
func TestMessageFilterMultiple(t *testing.T) {
	// 创建三个网络实例Node1、Node2和Node3，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)
	n3 := createTestNetwork(t, "127.0.0.1", 0)

	// 启动三个网络实例的运行上下文
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	defer cancel3()

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

	// 在Node2上注册过滤器：拒绝包含"node2-blocked"关键字的消息
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("node2-blocked"))
	})

	// 在Node3上注册过滤器：拒绝包含"node3-blocked"关键字的消息
	n3.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("node3-blocked"))
	})

	// 在Node2和Node3上注册消息处理器
	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播三种不同类型的消息
	node2BlockedMessage := []byte("message blocked by node2: node2-blocked")
	node3BlockedMessage := []byte("message blocked by node3: node3-blocked")
	normalMessage := []byte("normal message that should reach both nodes")

	// 广播被Node2过滤的消息
	err := n1.BroadcastMessage("test-topic", node2BlockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast node2-blocked message: %v", err)
	}

	// 广播被Node3过滤的消息
	err = n1.BroadcastMessage("test-topic", node3BlockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast node3-blocked message: %v", err)
	}

	// 广播正常消息
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2只接收到正常消息和被Node3过滤的消息（Node2会转发被Node3过滤的消息）
	messages2 := collector2.getMessages()
	expectedMessages2 := 2 // 正常消息 + 被Node3过滤的消息
	if len(messages2) != expectedMessages2 {
		t.Errorf("Node2 expected %d messages, got %d messages", expectedMessages2, len(messages2))
	}

	// 验证Node3只接收到正常消息和被Node2过滤的消息（Node3会接收到被Node2过滤的消息）
	messages3 := collector3.getMessages()
	expectedMessages3 := 2 // 正常消息 + 被Node2过滤的消息
	if len(messages3) != expectedMessages3 {
		t.Errorf("Node3 expected %d messages, got %d messages", expectedMessages3, len(messages3))
	}

	// 验证Node2没有接收到被自己过滤的消息
	for _, msg := range messages2 {
		if bytes.Contains(msg.Data, []byte("node2-blocked")) {
			t.Error("Node2 received message that should have been filtered by its own filter")
		}
	}

	// 验证Node3没有接收到被自己过滤的消息
	for _, msg := range messages3 {
		if bytes.Contains(msg.Data, []byte("node3-blocked")) {
			t.Error("Node3 received message that should have been filtered by its own filter")
		}
	}

	// 验证两个节点都接收到正常消息
	foundNormalInNode2 := false
	foundNormalInNode3 := false

	for _, msg := range messages2 {
		if bytes.Equal(msg.Data, normalMessage) {
			foundNormalInNode2 = true
			break
		}
	}

	for _, msg := range messages3 {
		if bytes.Equal(msg.Data, normalMessage) {
			foundNormalInNode3 = true
			break
		}
	}

	if !foundNormalInNode2 {
		t.Error("Node2 did not receive the normal message")
	}

	if !foundNormalInNode3 {
		t.Error("Node3 did not receive the normal message")
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}
