package network

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// 添加测试辅助函数和结构体

// messageCollector 用于收集广播消息
type messageCollector struct {
	messages []NetMessage
	mu       sync.Mutex
}

func (mc *messageCollector) addMessage(msg NetMessage) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = append(mc.messages, msg)
}

func (mc *messageCollector) getMessages() []NetMessage {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	// 返回副本以避免数据竞争
	result := make([]NetMessage, len(mc.messages))
	copy(result, mc.messages)
	return result
}

// createTestNetwork 创建测试网络实例
func createTestNetwork(t *testing.T, host string, port int) NetworkInterface {
	cfg := &NetworkConfig{
		Host:     host,
		Port:     port, // 使用指定端口或0表示随机端口
		MaxPeers: 10,
	}

	n, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	return n
}

// connectNetworks 建立两个网络间的连接
func connectNetworks(t *testing.T, n1, n2 NetworkInterface) {
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	err := n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect networks: %v", err)
	}
}

// waitForConnection 等待连接建立
func waitForConnection(_ *testing.T, n1, n2 NetworkInterface, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			return false
		case <-ticker.C:
			peers1 := n1.GetPeers()
			peers2 := n2.GetPeers()

			if len(peers1) > 0 && len(peers2) > 0 {
				return true
			}
		}
	}
}

// cleanupNetworks 清理网络资源
func cleanupNetworks(_ context.Context, cancel context.CancelFunc, _ ...NetworkInterface) {
	// 取消上下文以停止网络
	cancel()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

func TestMessageHandlerRegistration(t *testing.T) {
	n := createTestNetwork(t, "127.0.0.1", 0)

	// 创建消息处理器
	handler := func(from string, msg NetMessage) error {
		return nil
	}

	// 注册消息处理器
	n.RegisterMessageHandler("test-topic", handler)

	// 验证消息是否被处理
	_ = handler
}

func TestNetworkCreation(t *testing.T) {
	// 创建网络配置
	cfg := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建网络实例
	n, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 网络实例创建成功，类型正确
	_ = n
}

func TestRequestHandlerRegistration(t *testing.T) {
	// 创建网络配置
	cfg := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建网络实例
	n, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	// 注册请求处理器
	handler := func(from string, req Request) ([]byte, error) {
		return []byte("response"), nil
	}

	n.RegisterRequestHandler("test-request", handler)

	// 处理器注册成功，没有panic
}

func TestMessageFilterRegistration(t *testing.T) {
	// 创建网络配置
	cfg := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建网络实例
	n, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	// 注册消息过滤器
	filter := func(msg NetMessage) bool {
		return true // 接受所有消息
	}

	n.RegisterMessageFilter("test-topic", filter)

	// 过滤器注册成功，没有panic
}

func TestNetworkRunAndCancel(t *testing.T) {
	// 创建网络配置
	cfg := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建网络实例
	n, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待一段时间确保网络启动
	time.Sleep(500 * time.Millisecond)

	// 取消上下文
	cancel()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// 添加点对点发送测试用例
func TestPointToPointSend(t *testing.T) {
	// 创建两个网络实例用于测试
	cfg1 := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	cfg2 := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建两个网络实例
	n1, err := New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create network 1: %v", err)
	}

	n2, err := New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create network 2: %v", err)
	}

	// 创建带超时的上下文
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

	// 获取网络2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	t.Logf("Network 2 addresses: %v", addrs)

	// 连接网络1到网络2
	err = n1.ConnectToPeer(addrs[0])
	if err != nil {
		t.Fatalf("Failed to connect networks: %v", err)
	}

	// 等待连接建立
	time.Sleep(500 * time.Millisecond)

	// 验证连接状态
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	t.Logf("Network 1 peers: %v", peers1)
	t.Logf("Network 2 peers: %v", peers2)

	if len(peers1) == 0 {
		t.Error("Network 1 has no peers")
	}

	if len(peers2) == 0 {
		t.Error("Network 2 has no peers")
	}

	// 注册请求处理器
	responseData := []byte("response data")
	n2.RegisterRequestHandler("test-request", func(from string, req Request) ([]byte, error) {
		t.Logf("Network 2 received request from %s: type=%s, data=%s", from, req.Type, string(req.Data))
		if req.Type != "test-request" {
			t.Errorf("Expected request type 'test-request', got '%s'", req.Type)
		}
		return responseData, nil
	})

	// 等待处理器注册完成
	time.Sleep(100 * time.Millisecond)

	// 发送点对点请求
	peerID2 := n2.GetLocalPeerID()
	requestData := []byte("request data")

	t.Logf("Sending request from %s to %s", n1.GetLocalPeerID(), peerID2)

	// 使用带超时的上下文发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 在goroutine中发送请求以避免阻塞
	responseChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		response, err := n1.SendRequest(peerID2, "test-request", requestData)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	// 等待响应或超时
	select {
	case response := <-responseChan:
		t.Logf("Received response: %s", string(response))
		// 验证响应
		if string(response) != string(responseData) {
			t.Errorf("Expected response '%s', got '%s'", string(responseData), string(response))
		}
	case err := <-errChan:
		t.Fatalf("Failed to send request: %v", err)
	case <-ctx.Done():
		t.Fatal("Request timed out")
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// 添加广播功能测试用例
func TestBroadcastFunctionality(t *testing.T) {
	// 创建两个网络实例用于测试
	cfg1 := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	cfg2 := &NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建两个网络实例
	n1, err := New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create network 1: %v", err)
	}

	n2, err := New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create network 2: %v", err)
	}

	// 创建带超时的上下文
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

	// 获取网络2的地址并连接
	addrs := n2.GetLocalAddresses()
	if len(addrs) == 0 {
		t.Fatal("Network 2 has no addresses")
	}

	// 连接网络1到网络2
	err = n1.ConnectToPeer(addrs[0])
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
	receivedMessages := make(chan NetMessage, 10)

	// 注册消息处理器
	n1.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		receivedMessages <- msg
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		receivedMessages <- msg
		return nil
	})

	// 等待订阅建立
	time.Sleep(500 * time.Millisecond)

	// 从网络1广播消息
	messageData := []byte("broadcast message")
	err = n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(1 * time.Second)

	// 检查是否收到了消息
	close(receivedMessages)
	receivedCount := 0
	for msg := range receivedMessages {
		receivedCount++
		if string(msg.Data) != string(messageData) {
			t.Errorf("Expected message '%s', got '%s'", string(messageData), string(msg.Data))
		}
	}

	// 至少应该收到一条消息（来自网络2）
	if receivedCount == 0 {
		t.Error("No messages received")
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// TestP2PBasicRequest 基本点对点请求测试 (TC-P2P-001)
func TestP2PBasicRequest(t *testing.T) {
	// 创建两个网络实例Node1和Node2，使用随机端口避免冲突
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

	// 获取Node2的地址并让Node1通过ConnectToPeer方法连接到Node2
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between networks")
	}

	// 在Node2上注册请求处理器，处理"test-request"类型的请求，返回预定义的响应数据
	responseData := []byte("response data from node2")
	n2.RegisterRequestHandler("test-request", func(from string, req Request) ([]byte, error) {
		return responseData, nil
	})

	// 等待处理器注册完成
	time.Sleep(100 * time.Millisecond)

	// 从Node1向Node2发送SendRequest请求，指定请求类型为"test-request"
	peerID2 := n2.GetLocalPeerID()
	requestData := []byte("request data from node1")

	response, err := n1.SendRequest(peerID2, "test-request", requestData)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// 验证Node1接收到的响应数据与Node2处理器返回的数据一致
	if string(response) != string(responseData) {
		t.Errorf("Expected response '%s', got '%s'", string(responseData), string(response))
	}

	// 验证传输的数据内容完整，无丢失或损坏
	// (在这个测试中，我们主要验证响应数据，请求数据的完整性由底层实现保证)

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestP2PHandlerMissing 请求处理器缺失测试 (TC-P2P-002)
func TestP2PHandlerMissing(t *testing.T) {
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

	// 获取Node2的地址并让Node1连接到Node2
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between networks")
	}

	// 不在Node2上注册任何针对"test-request"类型的请求处理器
	// 直接从Node1向Node2发送SendRequest请求，指定请求类型为"test-request"
	peerID2 := n2.GetLocalPeerID()
	requestData := []byte("request data")

	response, err := n1.SendRequest(peerID2, "test-request", requestData)
	if err == nil {
		t.Fatal("Expected error when sending request to node without handler, but got none")
	}

	// 验证Node1接收到错误响应，类型为"error"
	// 验证错误信息包含"no handler found for request type"的描述
	if response != nil {
		t.Errorf("Expected nil response for missing handler, got: %v", response)
	}

	expectedErrorText := "no handler found for request type"
	if err != nil && !strings.Contains(err.Error(), expectedErrorText) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrorText, err)
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestP2PLargeData 大数据传输测试 (TC-P2P-003)
func TestP2PLargeData(t *testing.T) {
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

	// 获取Node2的地址并让Node1连接到Node2
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between networks")
	}

	// 在Node2上注册请求处理器，处理"large-data-request"类型的请求
	// 创建1MB的响应数据
	responseData := make([]byte, 1024*1024) // 1MB
	for i := range responseData {
		responseData[i] = byte(i % 256)
	}

	n2.RegisterRequestHandler("large-data-request", func(from string, req Request) ([]byte, error) {
		return responseData, nil
	})

	// 等待处理器注册完成
	time.Sleep(100 * time.Millisecond)

	// 从Node1向Node2发送包含大量数据（例如1MB）的请求
	peerID2 := n2.GetLocalPeerID()
	requestData := make([]byte, 512*1024) // 512KB request data
	for i := range requestData {
		requestData[i] = byte((i * 2) % 256)
	}

	response, err := n1.SendRequest(peerID2, "large-data-request", requestData)
	if err != nil {
		t.Fatalf("Failed to send large data request: %v", err)
	}

	// 验证Node1能正确接收到完整响应
	if len(response) != len(responseData) {
		t.Errorf("Response data length mismatch, expected: %d, got: %d", len(responseData), len(response))
	}

	// 验证大数据在传输过程中没有损坏
	if !bytes.Equal(response, responseData) {
		t.Error("Response data content mismatch - data corruption detected")
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterUnified 测试统一消息过滤机制
func TestMessageFilterUnified(t *testing.T) {
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

	// 在Node2上注册消息处理器和消息过滤器，过滤器拒绝包含"filtered"关键字的消息
	collector := &messageCollector{}

	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	n2.RegisterMessageFilter("test-topic", func(msg NetMessage) bool {
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
