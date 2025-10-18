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
	// Create network configuration, enable Peer scoring feature
	cfg := &network.NetworkConfig{
		Host:               "127.0.0.1",
		Port:               0, // Random port
		MaxPeers:           10,
		EnablePeerScoring:  true, // Enable Peer scoring
		IPColocationWeight: -0.1, // IP colocation weight
		BehaviourWeight:    -1.0, // Behavior weight
		BehaviourDecay:     0.98, // Behavior decay factor
	}

	// Create network instance
	net, err := network.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// Register broadcast message handler
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("[%s] Received message from %s: %s\n", time.Now().Format("15:04:05"), from[:8], string(msg.Data))
		return nil
	})

	// Register point-to-point request handler
	net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("[%s] Received echo request from %s: %s\n", time.Now().Format("15:04:05"), from[:8], string(req.Data))
		return req.Data, nil
	})

	// Print local node addresses
	fmt.Println("Local node addresses:")
	for _, addr := range net.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// Start network
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for system interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	fmt.Println("\nPeer scoring network is ready!")
	fmt.Println("Commands:")
	fmt.Println("  /connect <multiaddr>  - Connect to a peer")
	fmt.Println("  /peers               - List connected peers")
	fmt.Println("  /quit                - Exit the program")
	fmt.Println("  Any other text       - Broadcast message to chat")
	fmt.Println("")

	// Simulate some network activity
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Periodically broadcast message
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

	// Wait for user interrupt
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// Cancel context to stop network
	cancel()

	// Wait for network to fully stop
	time.Sleep(1 * time.Second)
	fmt.Println("Peer scoring example finished")
}
