package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lengzhao/network"
)

func main() {
	// 创建网络配置
	cfg := &network.NetworkConfig{
		Host:           "127.0.0.1",
		Port:           0,
		MaxPeers:       100,
		BootstrapPeers: []string{},
	}

	// 创建网络实例
	net, err := network.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// 注册聊天消息处理器
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[%s] %s: %s\n", time.Now().Format("15:04:05"), from, string(msg.Data))
		return nil
	})

	// 注册用户信息请求处理器
	net.RegisterRequestHandler("userinfo", func(from string, req network.Request) ([]byte, error) {
		userInfo := fmt.Sprintf("User: %s", from)
		fmt.Printf("[%s] Received userinfo request from %s\n", time.Now().Format("15:04:05"), from)
		return []byte(userInfo), nil
	})

	// 打印本地节点地址
	fmt.Println("Local node addresses:")
	for _, addr := range net.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// 监听系统中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动网络
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	fmt.Println("\nChat network is ready!")
	fmt.Println("Commands:")
	fmt.Println("  /connect <multiaddr>  - Connect to a peer")
	fmt.Println("  /peers               - List connected peers")
	fmt.Println("  /userinfo <peer_id>  - Get user info from a peer")
	fmt.Println("  /quit                - Exit the chat")
	fmt.Println("  Any other text       - Broadcast message to chat")
	fmt.Println("")

	// 读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if len(input) == 0 {
			continue
		}

		// 处理命令
		if strings.HasPrefix(input, "/") {
			parts := strings.Fields(input)
			command := parts[0]

			switch command {
			case "/connect":
				if len(parts) < 2 {
					fmt.Println("Usage: /connect <multiaddr>")
					continue
				}
				addr := parts[1]
				fmt.Printf("Connecting to %s...\n", addr)
				err := net.ConnectToPeer(addr)
				if err != nil {
					fmt.Printf("Failed to connect: %v\n", err)
				} else {
					fmt.Println("Connected successfully!")
				}

			case "/peers":
				peers := net.GetPeers()
				if len(peers) == 0 {
					fmt.Println("No connected peers")
				} else {
					fmt.Println("Connected peers:")
					for _, peer := range peers {
						fmt.Printf("  %s\n", peer)
					}
				}

			case "/userinfo":
				if len(parts) < 2 {
					fmt.Println("Usage: /userinfo <peer_id>")
					continue
				}
				peerID := parts[1]
				fmt.Printf("Requesting userinfo from %s...\n", peerID)
				response, err := net.SendRequest(peerID, "userinfo", []byte{})
				if err != nil {
					fmt.Printf("Failed to get userinfo: %v\n", err)
				} else {
					fmt.Printf("User info: %s\n", string(response))
				}

			case "/quit":
				fmt.Println("Shutting down...")
				goto shutdown

			default:
				fmt.Printf("Unknown command: %s\n", command)
			}
		} else {
			// 广播聊天消息
			err := net.BroadcastMessage("chat", []byte(input))
			if err != nil {
				fmt.Printf("Failed to broadcast message: %v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}

shutdown:
	// 取消上下文以停止网络
	cancel()

	// 等待网络完全停止
	time.Sleep(1 * time.Second)
	fmt.Println("Chat example finished")
}
