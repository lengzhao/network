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
	// 创建网络配置
	cfg := &network.NetworkConfig{
		Host:           "0.0.0.0",
		Port:           8000,
		MaxPeers:       100,
		PrivateKeyPath: "./private_key.pem",
		BootstrapPeers: []string{}, // 在实际应用中可以添加引导节点
	}

	// 创建网络实例
	net, err := network.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// 注册广播消息处理器
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("Received broadcast message from %s on topic %s: %s\n", from, msg.Topic, string(msg.Data))
		return nil
	})

	// 注册点对点请求处理器
	net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("Received request from %s: %s\n", from, string(req.Data))
		// 回显请求数据作为响应
		return req.Data, nil
	})

	// 注册消息过滤器（可选）
	net.RegisterMessageFilter("chat", func(msg network.NetMessage) bool {
		// 过滤掉包含"spam"的消息
		if string(msg.Data) == "spam" {
			fmt.Println("Filtered out spam message")
			return false
		}
		return true
	})

	// 打印本地节点地址
	fmt.Println("Local node addresses:")
	for _, addr := range net.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// 启动网络
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中运行网络
	go func() {
		if err := net.Run(ctx); err != nil {
			log.Printf("Network stopped with error: %v", err)
		} else {
			log.Println("Network stopped")
		}
	}()

	// 等待网络启动
	time.Sleep(1 * time.Second)

	// 演示广播消息
	fmt.Println("\n--- Broadcasting message ---")
	err = net.BroadcastMessage("chat", []byte("Hello, world!"))
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}

	// 演示点对点请求（需要有其他节点连接才能工作）
	fmt.Println("\n--- Sending request to self ---")
	// 获取本地Peer ID并发送请求给自己
	localPeerID := net.GetLocalPeerID()
	response, err := net.SendRequest(localPeerID, "echo", []byte("Hello, self!"))
	if err != nil {
		log.Printf("Failed to send request: %v", err)
	} else {
		fmt.Printf("Received response: %s\n", string(response))
	}

	// 等待用户中断或30秒后自动退出
	fmt.Println("\nNetwork is running. Press Ctrl+C to stop.")
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal, shutting down...")
	case <-time.After(30 * time.Second):
		fmt.Println("\nTimeout reached, shutting down...")
	}

	// 取消上下文以停止网络
	cancel()

	// 等待网络完全停止
	time.Sleep(1 * time.Second)
	fmt.Println("Example finished")
}