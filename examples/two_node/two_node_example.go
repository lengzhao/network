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
	// Create first node
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

	// Create second node
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

	// Register handlers for node 1
	net1.RegisterMessageHandler("chat", func(from string, topic string, data []byte) error {
		fmt.Printf("[Node 1] Received broadcast message from %s on topic %s: %s\n", from, topic, string(data))
		return nil
	})

	// Node 1 needs to handle echo and query requests from node 2
	net1.RegisterRequestHandler("echo", func(from string, reqType string, data []byte) ([]byte, error) {
		fmt.Printf("[Node 1] Received echo request from %s: %s\n", from, string(data))
		return append([]byte("Echo: "), data...), nil
	})

	net1.RegisterRequestHandler("query", func(from string, reqType string, data []byte) ([]byte, error) {
		fmt.Printf("[Node 1] Received query from %s: %s\n", from, string(data))
		return []byte("Current time: " + time.Now().Format(time.RFC3339)), nil
	})

	// Register handlers for node 2
	net2.RegisterMessageHandler("chat", func(from string, topic string, data []byte) error {
		fmt.Printf("[Node 2] Received broadcast message from %s on topic %s: %s\n", from, topic, string(data))
		return nil
	})

	// Node 2 needs to handle echo and query requests from node 1
	net2.RegisterRequestHandler("echo", func(from string, reqType string, data []byte) ([]byte, error) {
		fmt.Printf("[Node 2] Received echo request from %s: %s\n", from, string(data))
		return append([]byte("Echo: "), data...), nil
	})

	net2.RegisterRequestHandler("query", func(from string, reqType string, data []byte) ([]byte, error) {
		fmt.Printf("[Node 2] Received query from %s: %s\n", from, string(data))
		return []byte("Current time: " + time.Now().Format(time.RFC3339)), nil
	})

	// Start both nodes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start node 1
	go func() {
		fmt.Println("[Node 1] Starting...")
		if err := net1.Run(ctx); err != nil {
			log.Printf("[Node 1] Stopped with error: %v", err)
		} else {
			log.Println("[Node 1] Stopped")
		}
	}()

	// Start node 2
	go func() {
		fmt.Println("[Node 2] Starting...")
		if err := net2.Run(ctx); err != nil {
			log.Printf("[Node 2] Stopped with error: %v", err)
		} else {
			log.Println("[Node 2] Stopped")
		}
	}()

	// Wait for nodes to start and handler registration to complete
	time.Sleep(2 * time.Second)

	// Print node addresses
	fmt.Println("\n[Node 1] Addresses:")
	for _, addr := range net1.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	fmt.Println("\n[Node 2] Addresses:")
	for _, addr := range net2.GetLocalAddresses() {
		fmt.Printf("  %s\n", addr)
	}

	// Connect two nodes
	fmt.Println("\n--- Connecting nodes ---")
	// Use node 2's first address to connect to node 1
	if len(net2.GetLocalAddresses()) > 0 {
		addr2 := net2.GetLocalAddresses()[0]
		fmt.Printf("Connecting Node 1 to Node 2 at address: %s\n", addr2)
		err = net1.ConnectToPeer(addr2)
		if err != nil {
			log.Printf("Failed to connect nodes: %v", err)
		} else {
			fmt.Println("Nodes connected successfully")
		}
	}

	// Wait for connections and subscriptions to establish
	fmt.Println("Waiting for connections and subscriptions to establish...")
	time.Sleep(3 * time.Second)

	// Check connection status
	fmt.Println("\n--- Connection status ---")
	peers1 := net1.GetPeers()
	peers2 := net2.GetPeers()
	fmt.Printf("[Node 1] Connected peers: %v\n", peers1)
	fmt.Printf("[Node 2] Connected peers: %v\n", peers2)

	// Demonstrate broadcast message
	fmt.Println("\n--- Broadcasting message ---")
	err = net1.BroadcastMessage("chat", []byte("Hello from Node 1!"))
	if err != nil {
		log.Printf("Node 1 failed to broadcast message: %v", err)
	} else {
		fmt.Println("Node 1 broadcast message: 'Hello from Node 1!'")
	}

	err = net2.BroadcastMessage("chat", []byte("Hello from Node 2!"))
	if err != nil {
		log.Printf("Node 2 failed to broadcast message: %v", err)
	} else {
		fmt.Println("Node 2 broadcast message: 'Hello from Node 2!'")
	}

	// Wait for broadcast messages to be processed
	fmt.Println("Waiting for broadcast messages to be processed...")
	time.Sleep(2 * time.Second)

	// Demonstrate P2P requests
	fmt.Println("\n--- Sending P2P requests ---")
	// Node 1 sends echo request to node 2
	node2PeerID := net2.GetLocalPeerID()
	fmt.Printf("Node 1 sending echo request to Node 2 (Peer ID: %s)\n", node2PeerID)
	time.Sleep(1 * time.Second) // Ensure handler registration is complete
	response, err := net1.SendRequest(node2PeerID, "echo", []byte("Hello Node 2!"))
	if err != nil {
		log.Printf("Node 1 failed to send echo request to Node 2: %v", err)
	} else {
		fmt.Printf("[Node 1] Received echo response from Node 2: %s\n", string(response))
	}

	// Node 2 sends query request to node 1
	node1PeerID := net1.GetLocalPeerID()
	fmt.Printf("Node 2 sending query request to Node 1 (Peer ID: %s)\n", node1PeerID)
	time.Sleep(1 * time.Second) // Ensure handler registration is complete
	response, err = net2.SendRequest(node1PeerID, "query", []byte("What time is it?"))
	if err != nil {
		log.Printf("Node 2 failed to send query request to Node 1: %v", err)
	} else {
		fmt.Printf("[Node 2] Received query response from Node 1: %s\n", string(response))
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
