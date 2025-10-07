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
	// 创建第一个节点
	cfg1 := &network.NetworkConfig{
		Host:           "127.0.0.1",
		Port:           8001,
		MaxPeers:       100,
		PrivateKeyPath: "./node1_private_key.pem",
		BootstrapPeers: []string{},
	}

	net1, err := network.New(cfg1)
	if err != nil {
		log.Fatalf("Failed to create node 1: %v", err)
	}

	// 创建第二个节点
	cfg2 := &network.NetworkConfig{
		Host:           "127.0.0.1",
		Port:           8002,
		MaxPeers:       100,
		PrivateKeyPath: "./node2_private_key.pem",
		BootstrapPeers: []string{},
	}

	net2, err := network.New(cfg2)
	if err != nil {
		log.Fatalf("Failed to create node 2: %v", err)
	}

	// 为节点1注册处理器
	net1.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[Node 1] Received broadcast message from %s: %s\n", from, string(msg.Data))
		return nil
	})

	net1.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("[Node 1] Received request from %s: %s\n", from, string(req.Data))
		return append([]byte("Echo: "), req.Data...), nil
	})

	// 为节点2注册处理器
	net2.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[Node 2] Received broadcast message from %s: %s\n", from, string(msg.Data))
		return nil
	})

	net2.RegisterRequestHandler("query", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("[Node 2] Received query from %s: %s\n", from, string(req.Data))
		return []byte("Current time: " + time.Now().Format(time.RFC3339)), nil
	})

	// 启动两个节点
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动节点1
	go func() {
		fmt.Println("[Node 1] Starting...")
		if err := net1.Run(ctx); err != nil {
			log.Printf("[Node 1] Stopped with error: %v", err)
		} else {
			log.Println("[Node 1] Stopped")
		}
	}()

	// 启动节点2
	go func() {
		fmt.Println("[Node 2] Starting...")
		if err := net2.Run(ctx); err != nil {
			log.Printf("[Node 2] Stopped with error: %v", err)
		} else {
			log.Println("[Node 2] Stopped")
		}
	}()

	// 等待节点启动
	time.Sleep(2 * time.Second)

	// 打印节点地址
	fmt.Println("\n[Node 1] Addresses:")
	for _, addr := range net1.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	fmt.Println("\n[Node 2] Addresses:")
	for _, addr := range net2.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// 连接两个节点
	fmt.Println("\n--- Connecting nodes ---")
	// 使用节点1的第一个地址连接到节点2
	if len(net1.GetLocalAddresses()) > 0 {
		addr1 := net1.GetLocalAddresses()[0]
		err = net2.ConnectToPeer(addr1)
		if err != nil {
			log.Printf("Failed to connect nodes: %v", err)
		} else {
			fmt.Println("Nodes connected successfully")
		}
	}

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 演示广播消息
	fmt.Println("\n--- Broadcasting message ---")
	err = net1.BroadcastMessage("chat", []byte("Hello from Node 1!"))
	if err != nil {
		log.Printf("Node 1 failed to broadcast message: %v", err)
	}

	err = net2.BroadcastMessage("chat", []byte("Hello from Node 2!"))
	if err != nil {
		log.Printf("Node 2 failed to broadcast message: %v", err)
	}

	// 等待广播消息处理
	time.Sleep(1 * time.Second)

	// 演示点对点请求
	fmt.Println("\n--- Sending P2P requests ---")
	// 节点1向节点2发送echo请求
	node2PeerID := net2.GetLocalPeerID()
	response, err := net1.SendRequest(node2PeerID, "echo", []byte("Hello Node 2!"))
	if err != nil {
		log.Printf("Node 1 failed to send request to Node 2: %v", err)
	} else {
		fmt.Printf("[Node 1] Received response from Node 2: %s\n", string(response))
	}

	// 节点2向节点1发送query请求
	node1PeerID := net1.GetLocalPeerID()
	response, err = net2.SendRequest(node1PeerID, "query", []byte("What time is it?"))
	if err != nil {
		log.Printf("Node 2 failed to send request to Node 1: %v", err)
	} else {
		fmt.Printf("[Node 2] Received response from Node 1: %s\n", string(response))
	}

	// 等待用户中断或20秒后自动退出
	fmt.Println("\nNetwork is running. Press Ctrl+C to stop.")
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal, shutting down...")
	case <-time.After(20 * time.Second):
		fmt.Println("\nTimeout reached, shutting down...")
	}

	// 取消上下文以停止网络
	cancel()

	// 等待网络完全停止
	time.Sleep(1 * time.Second)
	fmt.Println("Example finished")
}