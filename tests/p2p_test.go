package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

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

// TestPointToPointSend 测试点对点消息发送功能
func TestPointToPointSend(t *testing.T) {
	// 创建两个网络实例
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

	// 注册请求处理器
	responseData := []byte("response data")
	n2.RegisterRequestHandler("test-request", func(from string, req network.Request) ([]byte, error) {
		// 验证请求数据
		expectedData := []byte("request data")
		if string(req.Data) != string(expectedData) {
			t.Errorf("Request data mismatch. Expected: %s, Got: %s", string(expectedData), string(req.Data))
		}

		return responseData, nil
	})

	// 等待处理器注册
	time.Sleep(100 * time.Millisecond)

	// 从Node1向Node2发送请求
	requestData := []byte("request data")
	peerID := n2.GetLocalPeerID()
	resp, err := n1.SendRequest(peerID, "test-request", requestData)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// 验证响应数据
	if string(resp) != string(responseData) {
		t.Errorf("Response data mismatch. Expected: %s, Got: %s", string(responseData), string(resp))
	}

	// 取消上下文以停止网络
	cancel1()
	cancel2()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}
