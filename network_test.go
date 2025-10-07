package network

import (
	"context"
	"testing"
	"time"
)

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
