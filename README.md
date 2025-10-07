# P2P Network Library

A modern P2P network library based on libp2p for Go applications.

## Features

- Node discovery and connection management
- Message broadcasting with topic-based pub/sub
- Point-to-point request/response communication
- Message filtering and content-based filtering
- Node whitelist for access control
- Extensible architecture

## Installation

```bash
go get github.com/lengzhao/network
```

## Usage

### Basic Example

```go
import "github.com/lengzhao/network"

// Create network configuration
cfg := &network.NetworkConfig{
    Host:     "127.0.0.1",
    Port:     8080,
    MaxPeers: 100,
}

// Create network instance
net, err := network.New(cfg)
if err != nil {
    log.Fatal(err)
}

// Run the network
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    if err := net.Run(ctx); err != nil {
        log.Fatal(err)
    }
}()

// Register message handler
net.RegisterMessageHandler("chat", func(from string, msg network.NetMessage) error {
    fmt.Printf("Received message from %s: %s\n", from, string(msg.Data))
    return nil
})

// Broadcast message
net.BroadcastMessage("chat", []byte("Hello, world!"))
```

### Message Filtering

The library supports message filtering at two levels:

1. **Content-based filtering**: Filter messages based on their content
2. **Node-based filtering**: Filter messages based on sender identity

```go
// Register content filter
net.RegisterMessageFilter("chat", func(msg network.NetMessage) bool {
    // Reject messages containing "spam"
    return !strings.Contains(string(msg.Data), "spam")
})
```

### Node Whitelist

Node whitelist functionality can be configured at the network level:

```go
cfg := &network.NetworkConfig{
    Host:          "127.0.0.1",
    Port:          8080,
    MaxPeers:      100,
    PeerWhitelist: []string{"peer-id-1", "peer-id-2"},
}
```

Note: The node whitelist feature uses libp2p's PeerFilter mechanism, which controls which nodes can participate in pub/sub topics. It does not prevent network-level connections between nodes.

### Extended Message Filtering

For more advanced filtering requirements, you can use the ExtendedMessageFilter:

```go
extendedFilter := &network.ExtendedMessageFilter{
    Whitelist: map[string]bool{
        "allowed-peer-id": true,
    },
    ContentFilter: func(msg network.NetMessage) bool {
        // Custom content filtering logic
        return !bytes.Contains(msg.Data, []byte("blocked"))
    },
}

net.RegisterExtendedMessageFilter("chat", extendedFilter)
```

## API Reference

### Network Interface

- `Run(ctx context.Context) error` - Start the network
- `BroadcastMessage(topic string, data []byte) error` - Broadcast message to topic
- `RegisterMessageHandler(topic string, handler MessageHandler)` - Register message handler
- `RegisterRequestHandler(requestType string, handler RequestHandler)` - Register request handler
- `SendRequest(peerID string, requestType string, data []byte) ([]byte, error)` - Send point-to-point request
- `ConnectToPeer(addr string) error` - Connect to peer
- `GetPeers() []string` - Get connected peers
- `GetLocalAddresses() []string` - Get local addresses
- `GetLocalPeerID() string` - Get local peer ID
- `RegisterMessageFilter(topic string, filter MessageFilter)` - Register message filter
- `RegisterExtendedMessageFilter(topic string, filter *ExtendedMessageFilter)` - Register extended message filter

## Testing

The library includes comprehensive tests for all functionality:

```bash
go test -v ./tests/...
```

## License

MIT