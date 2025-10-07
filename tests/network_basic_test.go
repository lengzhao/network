package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// messageCollector 用于收集广播消息
type messageCollector struct {
	messages []network.NetMessage
	mu       sync.Mutex
}

func (mc *messageCollector) addMessage(msg network.NetMessage) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = append(mc.messages, msg)
}

func (mc *messageCollector) getMessages() []network.NetMessage {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	// 返回副本以避免数据竞争
	result := make([]network.NetMessage, len(mc.messages))
	copy(result, mc.messages)
	return result
}

func (mc *messageCollector) clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = []network.NetMessage{}
}

// createTestNetwork 创建测试网络实例
func createTestNetwork(t *testing.T, host string, port int) network.NetworkInterface {
	cfg := &network.NetworkConfig{
		Host:     host,
		Port:     port,
		MaxPeers: 10,
	}

	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	return n
}

// connectNetworks 建立两个网络间的连接
func connectNetworks(t *testing.T, n1, n2 network.NetworkInterface) {
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
func waitForConnection(_ *testing.T, n1, n2 network.NetworkInterface, timeout time.Duration) bool {
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
func cleanupNetworks(_ context.Context, cancel context.CancelFunc, _ ...network.NetworkInterface) {
	// 取消上下文以停止网络
	cancel()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}

// TestNetworkCreation 测试网络创建功能
func TestNetworkCreation(t *testing.T) {
	// 创建网络配置
	cfg := &network.NetworkConfig{
		Host:     "127.0.0.1",
		Port:     0, // 使用随机端口
		MaxPeers: 10,
	}

	// 创建网络实例
	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 网络实例创建成功，类型正确
	_ = n
}

// TestNetworkRunAndCancel 测试网络运行和取消功能
func TestNetworkRunAndCancel(t *testing.T) {
	// 创建网络实例
	n := createTestNetwork(t, "127.0.0.1", 0)

	// 创建运行上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(500 * time.Millisecond)

	// 验证网络地址
	addresses := n.GetLocalAddresses()
	if len(addresses) == 0 {
		t.Error("Network has no local addresses")
	}

	// 取消上下文以停止网络
	cancel()

	// 等待网络关闭
	time.Sleep(500 * time.Millisecond)
}
