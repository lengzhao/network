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