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
	// Create network instance
	net, err := network.New(nil)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// Register broadcast message handler
	net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
		fmt.Printf("Received broadcast message from %s on topic %s: %s\n", from, msg.Topic, string(msg.Data))
		return nil
	})

	// Register point-to-point request handler
	net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
		fmt.Printf("Received request from %s: %s\n", from, string(req.Data))
		// Echo request data as response
		return req.Data, nil
	})

	// Register message filter (optional)
	net.RegisterMessageFilter("chat", func(msg network.NetMessage) bool {
		// Filter out messages containing "spam"
		if string(msg.Data) == "spam" {
			fmt.Println("Filtered out spam message")
			return false
		}
		return true
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

	// Demonstrate broadcast message
	fmt.Println("\n--- Broadcasting message ---")
	err = net.BroadcastMessage("chat", []byte("Hello, world!"))
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}

	// Demonstrate point-to-point request (requires other nodes to be connected to work)
	fmt.Println("\n--- Sending request to self ---")
	// Get local Peer ID and send request to self
	localPeerID := net.GetLocalPeerID()
	response, err := net.SendRequest(localPeerID, "echo", []byte("Hello, self!"))
	if err != nil {
		log.Printf("Failed to send request: %v", err)
	} else {
		fmt.Printf("Received response: %s\n", string(response))
	}

	// Wait for user interrupt or exit automatically after 30 seconds
	fmt.Println("\nNetwork is running. Press Ctrl+C to stop.")
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal, shutting down...")
	case <-time.After(30 * time.Second):
		fmt.Println("\nTimeout reached, shutting down...")
	}

	// Cancel context to stop network
	cancel()

	// Wait for network to fully stop
	time.Sleep(1 * time.Second)
	fmt.Println("Example finished")
}
