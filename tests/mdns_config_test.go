package tests

import (
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestMDNSEnabled 测试启用MDNS功能
func TestMDNSEnabled(t *testing.T) {
	// 创建启用MDNS的网络配置
	cfg := &network.NetworkConfig{
		Host:        "127.0.0.1",
		Port:        0, // 随机端口
		MaxPeers:    10,
		DisableMDNS: false, // 不禁用MDNS（即启用MDNS）
	}

	// 创建网络实例
	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network with MDNS enabled: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 启动网络
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(1 * time.Second)

	// 网络应该正常运行
	// 这个测试主要验证网络可以正常创建和启动，而不会因为MDNS配置而出现错误
}

// TestMDNSDisabled 测试禁用MDNS功能
func TestMDNSDisabled(t *testing.T) {
	// 创建禁用MDNS的网络配置
	cfg := &network.NetworkConfig{
		Host:        "127.0.0.1",
		Port:        0, // 随机端口
		MaxPeers:    10,
		DisableMDNS: true, // 禁用MDNS
	}

	// 创建网络实例
	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network with MDNS disabled: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 启动网络
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(1 * time.Second)

	// 网络应该正常运行
	// 这个测试主要验证网络可以正常创建和启动，即使禁用了MDNS
}

// TestMDNSDefault 测试默认MDNS配置（应该启用）
func TestMDNSDefault(t *testing.T) {
	// 创建使用默认网络配置的实例（应该启用MDNS）
	cfg := network.NewNetworkConfig()
	cfg.Host = "127.0.0.1"
	cfg.Port = 0 // 随机端口
	cfg.MaxPeers = 10
	// 不修改DisableMDNS，使用NewNetworkConfig设置的默认值(false)

	// 创建网络实例
	n, err := network.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create network with default MDNS config: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 启动网络
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(1 * time.Second)

	// 网络应该正常运行
}

// TestMDNSNilConfig 测试nil配置（应该启用MDNS）
func TestMDNSNilConfig(t *testing.T) {
	// 使用nil配置创建网络实例（应该使用默认配置并启用MDNS）
	n, err := network.New(nil)
	if err != nil {
		t.Fatalf("Failed to create network with nil config: %v", err)
	}

	if n == nil {
		t.Fatal("Network instance is nil")
	}

	// 启动网络
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 在goroutine中运行网络
	go func() {
		err := n.Run(ctx)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Network run failed with error: %v", err)
		}
	}()

	// 等待网络启动
	time.Sleep(1 * time.Second)

	// 网络应该正常运行
}
