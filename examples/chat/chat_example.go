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
	// Create network configuration
	cfg := &network.NetworkConfig{
		Host:           "127.0.0.1",
		Port:           0,
		MaxPeers:       100,
		BootstrapPeers: []string{},
	}

	// Create network instance
	net, err := network.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// Register chat message handler
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[%s] %s: %s\n", time.Now().Format("15:04:05"), from, string(msg.Data))
		return nil
	})

	// Register user info request handler
	net.RegisterRequestHandler("userinfo", func(from string, req network.Request) ([]byte, error) {
		userInfo := fmt.Sprintf("User: %s", from)
		fmt.Printf("[%s] Received userinfo request from %s\n", time.Now().Format("15:04:05"), from)
		return []byte(userInfo), nil
	})

	// Print local node addresses
	fmt.Println("Local node addresses:")
	for _, addr := range net.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// Listen for system interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start network
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run network in goroutine
	go func() {
		if err := net.Run(ctx); err != nil {
			log.Printf("Network stopped with error: %v", err)
		} else {
			log.Println("Network stopped")
		}
	}()

	// Wait for network to start
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
