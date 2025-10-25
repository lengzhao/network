package tests

import (
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

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
