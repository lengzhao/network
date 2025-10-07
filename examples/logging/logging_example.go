package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lengzhao/network"
	"github.com/lengzhao/network/log"
)

func main() {
	// Create network configuration with custom logging
	logConfig := &log.LogConfig{
		Level:      log.DEBUG,
		OutputFile: "", // Empty string means no file output (stdout only)
		Format:     log.TEXT,
		Modules: map[string]log.ModuleConfig{
			"network.pubsub":   {Level: log.DEBUG},
			"network.security": {Level: log.WARN},
			"network.request":  {Level: log.INFO},
		},
		Enabled: true,
	}

	cfg := &network.NetworkConfig{
		Host:           "127.0.0.1",
		Port:           0, // Use random port
		MaxPeers:       10,
		PrivateKeyPath: "", // Generate temporary key
		LogConfig:      logConfig,
	}

	// Create network instance
	net, err := network.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create network: %v\n", err)
		return
	}

	// Run the network in a separate goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := net.Run(ctx); err != nil {
			fmt.Printf("Network error: %v\n", err)
		}
	}()

	// Give the network time to start
	time.Sleep(100 * time.Millisecond)

	fmt.Println("Network started with custom logging configuration")

	// Register a message handler
	net.RegisterMessageHandler("example", func(from string, msg network.NetMessage) error {
		fmt.Printf("Received message from %s: %s\n", from, string(msg.Data))
		return nil
	})

	// Example of using the new logging approach
	// Using With() directly for more intuitive logging
	logger := net.GetLogManager().With("module", "example")
	logger.Info("This is an example log message using the new approach")

	// We can also chain With() calls
	requestLogger := logger.With("component", "handler")
	requestLogger.Debug("Processing request", "from", "example_peer")

	// Wait a bit to see some log output
	time.Sleep(2 * time.Second)

	// Cancel the context to stop the network
	cancel()

	// Wait for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	fmt.Println("Example completed")
}