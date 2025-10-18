# Universal Network Module Based on go-libp2p

## Overview

This is a universal network module based on go-libp2p, which can be used in various P2P application scenarios such as blockchain and chat systems. The module provides simple and easy-to-use APIs, supporting both broadcast mode and point-to-point request mode, with filtering functionality for broadcast mode.

## Features

- **Dual Communication Modes**: Supports broadcast mode (publish/subscribe) and point-to-point request/response mode
- **Easy to Use**: Provides clean API interfaces to reduce the complexity of P2P network programming
- **Message Filtering**: Broadcast mode supports message filtering functionality
- **Automatic Discovery**: Supports mDNS local node discovery (can be configured to enable/disable)
- **Secure Transport**: Based on libp2p secure transport protocols
- **Extensibility**: Modular design, easy to integrate into different types of projects

## Installation

```bash
go get github.com/lengzhao/network
```

## Quick Start

### 1. Create Network Instance

```go
import (
    "github.com/lengzhao/network"
)

// Method 1: Use NewNetworkConfig to create default configuration (recommended)
cfg := network.NewNetworkConfig()
cfg.Host = "0.0.0.0"
cfg.Port = 8000
cfg.MaxPeers = 100
cfg.PrivateKeyPath = "./private_key.pem"
cfg.BootstrapPeers = []string{} // Optional bootstrap nodes

// Create network instance
net, err := network.New(cfg)
if err != nil {
    log.Fatalf("Failed to create network: %v", err)
}
```

### 2. Register Message Handlers

```go
// Register broadcast message handler
net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
    fmt.Printf("Received broadcast message from %s: %s\n", from, string(msg.Data))
    return nil
})

// Register point-to-point request handler
net.RegisterRequestHandler("echo", func(from string, req network.Request) ([]byte, error) {
    // Echo request data as response
    return req.Data, nil
})
```

### 3. Register Message Filter (Optional)

```go
// Register message filter
net.RegisterMessageFilter("chat", func(msg network.NetMessage) bool {
    // Filter out messages containing "spam"
    if string(msg.Data) == "spam" {
        return false
    }
    return true
})
```

### 4. Start Network

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Run network in goroutine
go func() {
    if err := net.Run(ctx); err != nil {
        log.Printf("Network stopped with error: %v", err)
    }
}()

// Wait for network to start
time.Sleep(1 * time.Second)
```

### 5. Use Network Functions

```go
// Broadcast message
err = net.BroadcastMessage("chat", []byte("Hello, world!"))
if err != nil {
    log.Printf("Failed to broadcast message: %v", err)
}

// Send point-to-point request
response, err := net.SendRequest(targetPeerID, "echo", []byte("Hello!"))
if err != nil {
    log.Printf("Failed to send request: %v", err)
} else {
    fmt.Printf("Received response: %s\n", string(response))
}
```

## Configuration Options

### NetworkConfig Configuration Description

| Field Name | Type | Description | Default Value |
|------------|------|-------------|---------------|
| Host | string | Listening address | "0.0.0.0" |
| Port | int | Listening port | 0 |
| MaxPeers | int | Maximum number of connections | 100 |
| PrivateKeyPath | string | Private key file path | "" |
| BootstrapPeers | []string | Bootstrap node address list | Empty array |
| PeerWhitelist | []string | Node whitelist (list of node ID strings) | Empty array |
| LogConfig | *log.LogConfig | Log configuration | nil |
| DisableMDNS | bool | Whether to disable MDNS discovery | false (when created with NewNetworkConfig()) |
| EnablePeerScoring | bool | Whether to enable Peer scoring | true |
| DiscoveryInterval | time.Duration | Node discovery interval | 1 minute |
| MaxIPColocation | int | Maximum number of nodes per IP address | 3 |
| ScoreInspectInterval | time.Duration | Score inspection interval | 1 minute |
| IPColocationWeight | float64 | IP colocation weight | -0.1 |
| BehaviourWeight | float64 | Behavior weight | -1.0 |
| BehaviourDecay | float64 | Behavior decay factor | 0.98 |

**Note**: When directly creating a NetworkConfig struct, the zero value of DisableMDNS is false, meaning mDNS is enabled by default. It is recommended to use the network.NewNetworkConfig() function to create configuration instances to get correct default values.

## API Reference

### NetworkInterface Interface

```go
type NetworkInterface interface {
    // Run Start the network module and run
    Run(ctx context.Context) error

    // BroadcastMessage Broadcast message to specified topic
    BroadcastMessage(topic string, data []byte) error

    // RegisterMessageHandler Register broadcast message handler
    RegisterMessageHandler(topic string, handler MessageHandler)

    // RegisterRequestHandler Register point-to-point request handler
    RegisterRequestHandler(requestType string, handler RequestHandler)

    // SendRequest Send point-to-point request
    SendRequest(peerID string, requestType string, data []byte) ([]byte, error)

    // ConnectToPeer Connect to specified peer
    ConnectToPeer(addr string) error

    // GetPeers Get list of connected peers
    GetPeers() []string

    // GetLocalAddresses Get local peer address list
    GetLocalAddresses() []string

    // GetLocalPeerID Get local peer ID
    GetLocalPeerID() string

    // RegisterMessageFilter Register broadcast message filter
    RegisterMessageFilter(topic string, filter MessageFilter)
}
```

## Run Examples

```bash
# Run basic example
go run examples/base/basic_example.go

# Run two-node example
go run examples/two_node/two_node_example.go

# Run MDNS configuration example
go run examples/mdns/mdns_example.go
```

## Testing

```bash
# Run unit tests
go test -v ./...
```

## Dependencies

- [go-libp2p](https://github.com/libp2p/go-libp2p) - libp2p Go implementation
- [go-libp2p-pubsub](https://github.com/libp2p/go-libp2p-pubsub) - libp2p publish/subscribe system
- [go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) - Kademlia DHT implementation

## License

MIT