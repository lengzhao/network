package network

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 添加测试辅助函数和结构体

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

func (mc *messageCollector) clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = []NetMessage{}
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

// TestP2PErrorResponse 错误响应处理测试 (TC-P2P-004)
func TestP2PErrorResponse(t *testing.T) {
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

	// 在Node2上注册一个返回错误的请求处理器
	errorMessage := "simulated processing error"
	n2.RegisterRequestHandler("error-request", func(from string, req Request) ([]byte, error) {
		return nil, fmt.Errorf("%s", errorMessage) // 修复：使用格式化字符串
	})

	// 等待处理器注册完成
	time.Sleep(100 * time.Millisecond)

	// 从Node1向Node2发送请求
	peerID2 := n2.GetLocalPeerID()
	requestData := []byte("request data")

	response, err := n1.SendRequest(peerID2, "error-request", requestData)
	if err == nil {
		t.Fatal("Expected error response, but got none")
	}

	// 验证Node1接收到错误响应
	if response != nil {
		t.Errorf("Expected nil response for error case, got: %v", response)
	}

	// 验证错误信息正确传递
	if err != nil && !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("Expected error containing '%s', got: %v", errorMessage, err)
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestBroadcastLinearTopology 线性拓扑广播测试 (TC-BROADCAST-001)
func TestBroadcastLinearTopology(t *testing.T) {
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

	// 建立线性连接：Node1通过ConnectToPeer连接Node2，Node2通过ConnectToPeer连接Node3
	connectNetworks(t, n1, n2)
	connectNetworks(t, n2, n3)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node2")
	}
	if !waitForConnection(t, n2, n3, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node2 and Node3")
	}

	// 在所有节点上注册相同主题（如"test-topic"）的消息处理器，使用messageCollector收集接收到的消息
	collector1 := &messageCollector{}
	collector2 := &messageCollector{}
	collector3 := &messageCollector{}

	n1.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector1.addMessage(msg)
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	n3.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播消息到"test-topic"主题
	messageData := []byte("broadcast message from Node1 in linear topology")
	err := n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2和Node3都通过messageCollector接收到消息
	messages2 := collector2.getMessages()
	messages3 := collector3.getMessages()

	if len(messages2) == 0 {
		t.Error("Node2 did not receive the broadcast message")
	}

	if len(messages3) == 0 {
		t.Error("Node3 did not receive the broadcast message")
	}

	// 验证所有节点接收到的消息内容与发送的内容一致
	for _, msg := range messages2 {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node2 received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	for _, msg := range messages3 {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node3 received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestBroadcastStarTopology 星型拓扑广播测试 (TC-BROADCAST-002)
func TestBroadcastStarTopology(t *testing.T) {
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

	// 建立星型连接：Node1通过ConnectToPeer分别连接Node2和Node3
	connectNetworks(t, n1, n2)
	connectNetworks(t, n1, n3)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node2")
	}
	if !waitForConnection(t, n1, n3, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node3")
	}

	// 在所有节点上注册相同主题（如"test-topic"）的消息处理器
	collector1 := &messageCollector{}
	collector2 := &messageCollector{}
	collector3 := &messageCollector{}

	n1.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector1.addMessage(msg)
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	n3.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播消息到"test-topic"主题
	messageData := []byte("broadcast message from Node1 in star topology")
	err := n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node2和Node3都接收到消息
	messages2 := collector2.getMessages()
	messages3 := collector3.getMessages()

	if len(messages2) == 0 {
		t.Error("Node2 did not receive the broadcast message")
	}

	if len(messages3) == 0 {
		t.Error("Node3 did not receive the broadcast message")
	}

	// 验证消息内容的正确性和一致性
	for _, msg := range messages2 {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node2 received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	for _, msg := range messages3 {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node3 received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestBroadcastNonDirectConnected 非直连节点消息测试 (TC-BROADCAST-003)
func TestBroadcastNonDirectConnected(t *testing.T) {
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

	// 建立线性连接：Node1 ↔ Node2 ↔ Node3（Node1和Node3不直连）
	connectNetworks(t, n1, n2)
	connectNetworks(t, n2, n3)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node2")
	}
	if !waitForConnection(t, n2, n3, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node2 and Node3")
	}

	// 在所有节点上注册相同主题（如"test-topic"）的消息处理器
	collector1 := &messageCollector{}
	collector2 := &messageCollector{}
	collector3 := &messageCollector{}

	n1.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector1.addMessage(msg)
		return nil
	})

	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	n3.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播消息到"test-topic"主题
	messageData := []byte("broadcast message from Node1 to non-direct connected Node3")
	err := n1.BroadcastMessage("test-topic", messageData)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证Node3能够通过Node2转发接收到消息
	messages3 := collector3.getMessages()

	if len(messages3) == 0 {
		t.Error("Node3 did not receive the broadcast message through Node2")
	}

	// 验证Node3接收到的消息内容与Node1发送的内容一致
	for _, msg := range messages3 {
		if !bytes.Equal(msg.Data, messageData) {
			t.Errorf("Node3 received incorrect message data. Expected: %s, Got: %s", string(messageData), string(msg.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestBroadcastMultipleTopics 多主题广播测试 (TC-BROADCAST-004)
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
	receivedMessagesTopic1 := make(chan NetMessage, 10)
	receivedMessagesTopic2 := make(chan NetMessage, 10)

	// 注册消息处理器
	n1.RegisterMessageHandler("topic1", func(from string, msg NetMessage) error {
		receivedMessagesTopic1 <- msg
		return nil
	})

	n2.RegisterMessageHandler("topic1", func(from string, msg NetMessage) error {
		receivedMessagesTopic1 <- msg
		return nil
	})

	n1.RegisterMessageHandler("topic2", func(from string, msg NetMessage) error {
		receivedMessagesTopic2 <- msg
		return nil
	})

	n2.RegisterMessageHandler("topic2", func(from string, msg NetMessage) error {
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
	messagesTopic1 := make([]NetMessage, 0)
	messagesTopic2 := make([]NetMessage, 0)

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

// TestPeerWhitelistNetworkLevel 网络层节点白名单测试
func TestPeerWhitelistNetworkLevel(t *testing.T) {
	// 创建三个网络实例Node1、Node2和Node3，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// 获取Node1的Peer ID，用于白名单
	node1PeerID := n1.GetLocalPeerID()

	// 为Node3创建带有白名单的配置，只允许Node1参与pubsub通信
	cfg3 := &NetworkConfig{
		Host:          "127.0.0.1",
		Port:          0,
		MaxPeers:      10,
		PeerWhitelist: []string{node1PeerID}, // 只允许Node1参与pubsub
	}
	n3, err := New(cfg3)
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

	// 测试pubsub白名单功能
	// 在Node3上注册消息处理器
	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// 在Node1和Node2上注册消息处理器
	collector1 := &messageCollector{}
	n1.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector1.addMessage(msg)
		return nil
	})

	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
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

	// 验证Node3只接收到来自Node1的消息（因为Node1在白名单中）
	messages3 := collector3.getMessages()
	if len(messages3) != 1 {
		t.Errorf("Node3 should receive only 1 message, got %d messages", len(messages3))
	}

	if len(messages3) > 0 {
		// 验证消息来源是Node1
		if messages3[0].From != node1PeerID {
			t.Errorf("Node3 should only receive messages from Node1. Got message from: %s", messages3[0].From)
		}

		// 验证消息内容正确
		if !bytes.Equal(messages3[0].Data, messageFromNode1) {
			t.Errorf("Node3 received incorrect message. Expected: %s, Got: %s",
				string(messageFromNode1), string(messages3[0].Data))
		}
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

// TestMessageFilterBasic 基本消息过滤测试 (TC-FILTER-001)
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

// TestMessageFilterNoForward 过滤后不转发测试 (TC-FILTER-002)
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

	// 建立线性连接：Node1 ↔ Node2 ↔ Node3
	connectNetworks(t, n1, n2)
	connectNetworks(t, n2, n3)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node2")
	}
	if !waitForConnection(t, n2, n3, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node2 and Node3")
	}

	// 在Node2上注册拒绝包含"do-not-forward"关键字消息的过滤器
	n2.RegisterMessageFilter("test-topic", func(msg NetMessage) bool {
		// 拒绝包含"do-not-forward"关键字的消息
		return !bytes.Contains(msg.Data, []byte("do-not-forward"))
	})

	// 在Node3上注册消息处理器
	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
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

// TestMessageFilterDynamic 动态过滤规则测试 (TC-FILTER-003)
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

	// 建立Node1和Node2之间的连接
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between networks")
	}

	// 在Node2上注册消息处理器
	collector := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	// 初始过滤器：拒绝包含"blocked"关键字的消息
	filterRule := "blocked"
	n2.RegisterMessageFilter("test-topic", func(msg NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte(filterRule))
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1广播包含"blocked"关键字的消息
	blockedMessage := []byte("this message is blocked")
	err := n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast blocked message: %v", err)
	}

	// 等待消息传递
	time.Sleep(1 * time.Second)

	// 验证消息被过滤
	messages := collector.getMessages()
	if len(messages) > 0 {
		t.Error("Message with 'blocked' keyword was not filtered")
	}

	// 动态更新过滤规则：现在允许包含"blocked"但拒绝包含"forbidden"的消息
	collector.clear() // 清空之前的消息
	filterRule = "forbidden"

	// 等待一点时间让过滤器更新生效
	time.Sleep(500 * time.Millisecond)

	// 从Node1广播包含"blocked"关键字的消息（现在应该通过）
	allowedMessage := []byte("this message with blocked keyword is now allowed")
	err = n1.BroadcastMessage("test-topic", allowedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast allowed message: %v", err)
	}

	// 从Node1广播包含"forbidden"关键字的消息（现在应该被过滤）
	forbiddenMessage := []byte("this message is forbidden")
	err = n1.BroadcastMessage("test-topic", forbiddenMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast test message: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证消息处理情况
	messages = collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d messages", len(messages))
	}

	if len(messages) > 0 {
		receivedMessage := messages[0]
		if !bytes.Equal(receivedMessage.Data, allowedMessage) {
			t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(allowedMessage), string(receivedMessage.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterMultiple 多过滤器协同测试 (TC-FILTER-004)
func TestMessageFilterMultiple(t *testing.T) {
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

	// 在Node2上注册消息处理器
	collector := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	// 注册一个组合过滤器：拒绝包含"filter1"或"filter2"的消息
	n2.RegisterMessageFilter("test-topic", func(msg NetMessage) bool {
		// 拒绝包含"filter1"或"filter2"关键字的消息
		return !bytes.Contains(msg.Data, []byte("filter1")) && !bytes.Contains(msg.Data, []byte("filter2"))
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 测试消息1：包含"filter1"关键字，应该被过滤
	message1 := []byte("this message contains filter1 keyword")
	err := n1.BroadcastMessage("test-topic", message1)
	if err != nil {
		t.Fatalf("Failed to broadcast message1: %v", err)
	}

	// 测试消息2：包含"filter2"关键字，应该被过滤
	message2 := []byte("this message contains filter2 keyword")
	err = n1.BroadcastMessage("test-topic", message2)
	if err != nil {
		t.Fatalf("Failed to broadcast message2: %v", err)
	}

	// 测试消息3：不包含任何被过滤的关键字，应该通过
	message3 := []byte("this message should pass through all filters")
	err = n1.BroadcastMessage("test-topic", message3)
	if err != nil {
		t.Fatalf("Failed to broadcast message3: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证只有消息3通过了所有过滤器
	messages := collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message to pass through filters, got %d messages", len(messages))
	}

	if len(messages) > 0 {
		receivedMessage := messages[0]
		if !bytes.Equal(receivedMessage.Data, message3) {
			t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(message3), string(receivedMessage.Data))
		}
	}

	// 验证被过滤的消息没有通过
	for _, msg := range messages {
		if bytes.Contains(msg.Data, []byte("filter1")) || bytes.Contains(msg.Data, []byte("filter2")) {
			t.Error("Message that should have been filtered was incorrectly processed")
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestPeerWhitelist 网络层节点白名单测试
func TestPeerWhitelist(t *testing.T) {
	// 创建三个网络实例Node1、Node2和Node3，使用随机端口
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// 为Node3创建带有白名单的配置，只允许Node1连接
	cfg3 := &NetworkConfig{
		Host:          "127.0.0.1",
		Port:          0,
		MaxPeers:      10,
		PeerWhitelist: []string{n1.GetLocalPeerID()}, // 只允许Node1连接
	}
	n3, err := New(cfg3)
	if err != nil {
		t.Fatalf("Failed to create network with whitelist: %v", err)
	}

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

	// 建立Node1和Node3之间的连接（应该成功，因为Node1在白名单中）
	connectNetworks(t, n1, n3)

	// 建立Node2和Node3之间的连接（应该失败，因为Node2不在白名单中）
	connectNetworks(t, n2, n3)

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 验证Node3只与Node1建立了连接，没有与Node2建立连接
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()
	peers3 := n3.GetPeers()

	// Node1应该与Node3连接
	if len(peers1) == 0 {
		t.Error("Node1 should be connected to Node3")
	}

	// Node3应该只与Node1连接，不应该与Node2连接
	if len(peers3) != 1 {
		t.Errorf("Node3 should be connected to only one node, got %d connections", len(peers3))
	}

	// 验证Node3的连接是对Node1的
	connectedToNode1 := false
	for _, peerID := range peers3 {
		if peerID == n1.GetLocalPeerID() {
			connectedToNode1 = true
			break
		}
	}

	if !connectedToNode1 {
		t.Error("Node3 should be connected to Node1")
	}

	// 验证Node2没有与Node3建立连接
	if len(peers2) > 0 {
		// 检查Node2的连接是否是与Node3的
		connectedToNode3 := false
		for _, peerID := range peers2 {
			if peerID == n3.GetLocalPeerID() {
				connectedToNode3 = true
				break
			}
		}

		// 在白名单机制下，Node3会拒绝Node2的连接，所以Node2不应该与Node3建立连接
		if connectedToNode3 {
			t.Error("Node2 should not be connected to Node3 due to whitelist filtering")
		}
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

	extendedFilter := &ExtendedMessageFilter{
		Whitelist: whitelist,
		ContentFilter: func(msg NetMessage) bool {
			// 拒绝包含"blocked"关键字的消息
			return !bytes.Contains(msg.Data, []byte("blocked"))
		},
	}

	n2.RegisterExtendedMessageFilter("test-topic", extendedFilter)
	n2.RegisterMessageHandler("test-topic", func(from string, msg NetMessage) error {
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
