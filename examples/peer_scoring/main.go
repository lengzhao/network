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
	// 创建网络配置，启用Peer评分功能
	cfg := &network.NetworkConfig{
		Host:               "127.0.0.1",
		Port:               0, // 随机端口
		MaxPeers:           10,
		EnablePeerScoring:  true, // 启用Peer评分
		IPColocationWeight: -0.1, // IP共置权重
		BehaviourWeight:    -1.0, // 行为权重
		BehaviourDecay:     0.98, // 行为衰减因子
	}

	// 创建网络实例
	net, err := network.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// 注册广播消息处理器
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[%s] Received message from %s: %s\n", time.Now().Format("15:04:05"), from[:8], string(msg.Data))
		return nil
	})

	// 注册点对点请求处理器
	net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("[%s] Received echo request from %s: %s\n", time.Now().Format("15:04:05"), from[:8], string(req.Data))
		return req.Data, nil
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

	fmt.Println("\nPeer scoring network is ready!")
	fmt.Println("Commands:")
	fmt.Println("  /connect <multiaddr>  - Connect to a peer")
	fmt.Println("  /peers               - List connected peers")
	fmt.Println("  /quit                - Exit the program")
	fmt.Println("  Any other text       - Broadcast message to chat")
	fmt.Println("")

	// 模拟一些网络活动
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 定期广播消息
				message := fmt.Sprintf("Periodic message at %s", time.Now().Format("15:04:05"))
				err := net.BroadcastMessage("chat", []byte(message))
				if err != nil {
					fmt.Printf("Failed to broadcast message: %v\n", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待用户中断
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// 取消上下文以停止网络
	cancel()

	// 等待网络完全停止
	time.Sleep(1 * time.Second)
	fmt.Println("Peer scoring example finished")
}
