# P2P Network Library

A modern P2P network library based on libp2p for Go applications.

## Features

- Node discovery and connection management
- Message broadcasting with topic-based pub/sub
- Point-to-point request/response communication
- Message filtering and content-based filtering
- Node whitelist for access control
- Extensible architecture
- Flexible logging system based on Go's slog package

## Installation

```bash
go get github.com/lengzhao/network
```

## Usage

### Basic Example

```go
import "github.com/lengzhao/network"

// Create network configuration with logging
logConfig := &network.LogConfig{
    Level:      network.INFO,
    OutputFile: "/var/log/network.log",
    Format:     network.JSON,
    Modules: map[string]network.ModuleConfig{
        "network.pubsub": {Level: network.DEBUG},
        "network.security": {Level: network.WARN},
    },
    Enabled: true,
}

cfg := &network.NetworkConfig{
    Host:     "127.0.0.1",
    Port:     8080,
    MaxPeers: 100,
    LogConfig: logConfig,
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

// Use the new simplified logging approach
logger := net.GetLogManager().With("module", "chat")
logger.Info("Chat handler initialized")
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

### Logging

The library includes a flexible logging system based on Go's slog package. You can use it in two ways:

1. **Using GetLogger() (deprecated but still supported)**:
```go
logger := net.GetLogManager().GetLogger("mymodule")
logger.Info("This is a log message")
```

2. **Using With() method (recommended)**:
```go
logger := net.GetLogManager().With("module", "mymodule")
logger.Info("This is a log message")

// You can also chain With() calls
handlerLogger := logger.With("component", "handler")
handlerLogger.Debug("Processing request", "from", peerID)
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
- `GetLogManager() log.LogManager` - Get the log manager

### Logging Configuration

The library includes a flexible logging system based on Go's slog package:

- `LogConfig` - Main logging configuration structure
- `ModuleConfig` - Module-specific logging configuration
- Log levels: DEBUG, INFO, WARN, ERROR, CRITICAL
- Output formats: TEXT, JSON
- Module-specific prefixes and levels

## Testing

The library includes comprehensive tests for all functionality:

```bash
go test -v ./tests/...
```

Unit tests for the logging system:

```bash
go test -v ./log/...
```

## License

MIT