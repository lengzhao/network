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
func waitForConnection(t *testing.T, n1, n2 NetworkInterface, timeout time.Duration) bool {
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

// responseTracker 用于跟踪点对点请求的响应
type responseTracker struct {
	responses [][]byte
	errors    []error
	mu        sync.Mutex
}

func (rt *responseTracker) addResponse(response []byte, err error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if err != nil {
		rt.errors = append(rt.errors, err)
	} else {
		rt.responses = append(rt.responses, response)
	}
}

func (rt *responseTracker) getResponseCount() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return len(rt.responses)
}

func (rt *responseTracker) getErrorCount() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return len(rt.errors)
}

// waitForMessage 等待消息处理完成
func waitForMessage(t *testing.T, ch chan bool, timeout time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

// cleanupNetworks 清理网络资源
func cleanupNetworks(ctx context.Context, cancel context.CancelFunc, networks ...NetworkInterface) {
	// 取消上下文以停止网络
	cancel()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
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

func TestMessageSerialization(t *testing.T) {
	// 测试请求序列化和反序列化
	req := &Request{
		Type: "test",
		Data: []byte("test data"),
	}

	data, err := req.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data is empty")
	}

	// 测试反序列化
	newReq := &Request{}
	err = newReq.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize request: %v", err)
	}

	if newReq.Type != req.Type {
		t.Errorf("Request type mismatch, expected: %s, got: %s", req.Type, newReq.Type)
	}

	if string(newReq.Data) != string(req.Data) {
		t.Errorf("Request data mismatch, expected: %s, got: %s", string(req.Data), string(newReq.Data))
	}

	// 测试响应序列化和反序列化
	resp := &Response{
		Type: "test",
		Data: []byte("response data"),
	}

	respData, err := resp.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize response: %v", err)
	}

	if len(respData) == 0 {
		t.Error("Serialized response data is empty")
	}

	// 测试响应反序列化
	newResp := &Response{}
	err = newResp.Deserialize(respData)
	if err != nil {
		t.Fatalf("Failed to deserialize response: %v", err)
	}

	if newResp.Type != resp.Type {
		t.Errorf("Response type mismatch, expected: %s, got: %s", resp.Type, newResp.Type)
	}

	if string(newResp.Data) != string(resp.Data) {
		t.Errorf("Response data mismatch, expected: %s, got: %s", string(resp.Data), string(newResp.Data))
	}
}

func TestMessageHandlerRegistration(t *testing.T) {
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

	// 注册消息处理器
	handler := func(from string, msg NetMessage) error {
		return nil
	}

	n.RegisterMessageHandler("test-topic", handler)

	// 处理器注册成功，没有panic
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

	// 建立Node1和Node2之间的连接
	connectNetworks(t, n1, n2)

	// 等待连接建立
	if !waitForConnection(t, n1, n2, 5*time.Second) {
		t.Fatal("Failed to establish connection between Node1 and Node2")
	}

	// 在两个节点上分别注册不同主题（如"topic-a"和"topic-b"）的消息处理器
	collector1A := &messageCollector{}
	collector1B := &messageCollector{}
	collector2A := &messageCollector{}
	collector2B := &messageCollector{}

	// Node1处理器
	n1.RegisterMessageHandler("topic-a", func(from string, msg NetMessage) error {
		collector1A.addMessage(msg)
		return nil
	})
	n1.RegisterMessageHandler("topic-b", func(from string, msg NetMessage) error {
		collector1B.addMessage(msg)
		return nil
	})

	// Node2处理器
	n2.RegisterMessageHandler("topic-a", func(from string, msg NetMessage) error {
		collector2A.addMessage(msg)
		return nil
	})
	n2.RegisterMessageHandler("topic-b", func(from string, msg NetMessage) error {
		collector2B.addMessage(msg)
		return nil
	})

	// 等待订阅建立
	time.Sleep(1 * time.Second)

	// 从Node1向两个不同主题分别广播消息
	messageDataA := []byte("message for topic-a")
	messageDataB := []byte("message for topic-b")

	err := n1.BroadcastMessage("topic-a", messageDataA)
	if err != nil {
		t.Fatalf("Failed to broadcast message to topic-a: %v", err)
	}

	err = n1.BroadcastMessage("topic-b", messageDataB)
	if err != nil {
		t.Fatalf("Failed to broadcast message to topic-b: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证每个节点只接收到对应主题的消息
	messages1A := collector1A.getMessages()
	messages1B := collector1B.getMessages()
	messages2A := collector2A.getMessages()
	messages2B := collector2B.getMessages()

	// 验证Node1接收到topic-a的消息但没有接收到topic-b的消息（因为是自己发送的）
	if len(messages1A) > 0 {
		t.Error("Node1 should not receive its own messages")
	}

	if len(messages1B) > 0 {
		t.Error("Node1 should not receive its own messages")
	}

	// 验证Node2接收到两个主题的消息
	if len(messages2A) == 0 {
		t.Error("Node2 did not receive message for topic-a")
	}

	if len(messages2B) == 0 {
		t.Error("Node2 did not receive message for topic-b")
	}

	// 验证消息内容的正确性
	for _, msg := range messages2A {
		if !bytes.Equal(msg.Data, messageDataA) {
			t.Errorf("Node2 received incorrect message data for topic-a. Expected: %s, Got: %s", string(messageDataA), string(msg.Data))
		}
	}

	for _, msg := range messages2B {
		if !bytes.Equal(msg.Data, messageDataB) {
			t.Errorf("Node2 received incorrect message data for topic-b. Expected: %s, Got: %s", string(messageDataB), string(msg.Data))
		}
	}

	// 清理资源
	cleanupNetworks(context.Background(), cancel1, n1, n2)
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
