package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lengzhao/network"
)

func main() {
	// 创建启用MDNS的网络配置（默认情况）
	cfgWithMDNS := &network.NetworkConfig{
		Host:           "0.0.0.0",
		Port:           8001,
		MaxPeers:       100,
		PrivateKeyPath: "./private_key_with_mdns.pem",
		BootstrapPeers: []string{},
		DisableMDNS:    false, // 不禁用MDNS发现功能（默认启用）
	}

	// 创建禁用MDNS的网络配置
	cfgWithoutMDNS := &network.NetworkConfig{
		Host:           "0.0.0.0",
		Port:           8002,
		MaxPeers:       100,
		PrivateKeyPath: "./private_key_without_mdns.pem",
		BootstrapPeers: []string{},
		DisableMDNS:    true, // 禁用MDNS发现功能
	}

	fmt.Println("Creating network with MDNS enabled...")
	netWithMDNS, err := network.New(cfgWithMDNS)
	if err != nil {
		log.Fatalf("Failed to create network with MDNS: %v", err)
	}

	fmt.Println("Creating network with MDNS disabled...")
	netWithoutMDNS, err := network.New(cfgWithoutMDNS)
	if err != nil {
		log.Fatalf("Failed to create network without MDNS: %v", err)
	}

	// 注册消息处理器
	netWithMDNS.RegisterMessageHandler("test", func(from string, msg network.NetMessage) error {
		fmt.Printf("[MDNS Enabled] Received message from %s: %s\n", from, string(msg.Data))
		return nil
	})

	netWithoutMDNS.RegisterMessageHandler("test", func(from string, msg network.NetMessage) error {
		fmt.Printf("[MDNS Disabled] Received message from %s: %s\n", from, string(msg.Data))
		return nil
	})

	// 启动网络
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中运行两个网络
	go func() {
		fmt.Println("Starting network with MDNS enabled...")
		if err := netWithMDNS.Run(ctx); err != nil {
			log.Printf("Network with MDNS stopped with error: %v", err)
		} else {
			log.Println("Network with MDNS stopped")
		}
	}()

	go func() {
		fmt.Println("Starting network with MDNS disabled...")
		if err := netWithoutMDNS.Run(ctx); err != nil {
			log.Printf("Network without MDNS stopped with error: %v", err)
		} else {
			log.Println("Network without MDNS stopped")
		}
	}()

	// 等待网络启动
	time.Sleep(2 * time.Second)

	fmt.Println("\nNetworks are running. Check the logs to see the difference in MDNS behavior.")
	fmt.Println("Press Ctrl+C to stop.")

	// 等待用户中断
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// 取消上下文以停止网络
	cancel()

	// 等待网络完全停止
	time.Sleep(1 * time.Second)
	fmt.Println("MDNS example finished")
}
