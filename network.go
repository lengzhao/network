package network

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

// PeerScoreInspector Peer score inspector interface
// type PeerScoreInspector func(map[peer.ID]*pubsub.PeerScoreSnapshot) // Moved to types.go

// Network network layer - modern P2P network based on libp2p
type Network struct {
	config *NetworkConfig
	host   host.Host
	ctx    context.Context
	cancel context.CancelFunc

	// DHT automatic management (automatically handled by libp2p.Routing)

	// Gossipsub message propagation
	pubsub   *pubsub.PubSub
	topics   map[string]*pubsub.Topic
	topicsMu sync.RWMutex

	// Message processing
	messageHandlers map[string]MessageHandler
	handlersMu      sync.RWMutex

	// Message filtering
	messageFilters map[string]MessageFilter
	filtersMu      sync.RWMutex

	// Point-to-point request processing
	requestHandlers map[string]RequestHandler
	requestMu       sync.RWMutex

	// Node discovery
	mdnsService mdns.Service
}

// topicValidatorHandler topic validator handler function
func (n *Network) topicValidatorHandler(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	topicName := msg.GetTopic()

	// Check if there is a registered MessageFilter
	n.filtersMu.RLock()
	filter, exists := n.messageFilters[topicName]
	n.filtersMu.RUnlock()

	// Apply filter if MessageFilter exists
	if exists {
		// Reject message if filter returns false
		if !filter(msg.ReceivedFrom.String(), topicName, msg.Data) {
			return pubsub.ValidationReject // Reject message
		}
	}

	// Accept message by default
	return pubsub.ValidationAccept
}

// autoRegisterTopicValidator automatically register topic validator when joining pubsub
func (n *Network) autoRegisterTopicValidator(topic string) error {
	return n.pubsub.RegisterTopicValidator(topic, n.topicValidatorHandler)
}

var _ NetworkInterface = (*Network)(nil)

// New create a new network instance
func New(cfg *NetworkConfig, ops ...libp2p.Option) (NetworkInterface, error) {
	// If config is nil, use default configuration
	if cfg == nil {
		cfg = NewNetworkConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Get private key
	priv, err := getOrGeneratePrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to get private key", "error", err)
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Create connection manager
	connManager, err := connmgr.NewConnManager(
		20,                                     // Minimum number of connections (default value)
		cfg.MaxPeers,                           // Maximum number of connections
		connmgr.WithGracePeriod(time.Minute*5), // Grace period
	)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to create connection manager", "error", err)
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Parse bootstrap nodes
	var bootstrapPeers []peer.AddrInfo
	for _, peerAddr := range cfg.BootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			slog.With("module", "network").Warn("failed to parse bootstrap peer address", "address", peerAddr, "error", err)
			continue
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			slog.With("module", "network").Warn("failed to parse bootstrap peer info", "address", peerAddr, "error", err)
			continue
		}

		bootstrapPeers = append(bootstrapPeers, *addrInfo)
	}
	// if len(bootstrapPeers) == 0 {
	// 	bootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()
	// }
	options := []libp2p.Option{
		// Use generated key pair
		libp2p.Identity(priv),
		// Listening address
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Host, cfg.Port),         // TCP connection
			fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", cfg.Host, cfg.Port), // QUIC transport
		),
		// Support noise connection
		libp2p.Security(noise.ID, noise.New),
		// Support default transport protocols
		libp2p.DefaultTransports,
		// Try to use uPNP to open ports for NAT hosts
		libp2p.NATPortMap(),
		// Enable NAT service to help other nodes detect NAT
		libp2p.EnableNATService(),
		// Enable automatic relay
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{}),
		// Enable hole punching
		libp2p.EnableHolePunching(),
		// Use connection manager
		libp2p.ConnectionManager(connManager),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dhtInstance, err := dht.New(context.Background(), h, dht.BootstrapPeers(bootstrapPeers...))
			return dhtInstance, err
		}),
	}
	options = append(options, ops...)

	// Create libp2p host
	host, err := libp2p.New(
		options...,
	)
	if err != nil {
		cancel()
		slog.With("module", "network").Error("failed to create libp2p host", "error", err)
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	n := &Network{
		config:          cfg,
		host:            host,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string]MessageHandler),
		messageFilters:  make(map[string]MessageFilter),
		requestHandlers: make(map[string]RequestHandler),
		topics:          make(map[string]*pubsub.Topic),
	}

	// DHT automatically initialized by libp2p.Routing

	// Initialize Gossipsub
	if err := n.initializeGossipsub(); err != nil {
		cancel()
		slog.With("module", "network").Error("failed to initialize Gossipsub", "error", err)
		return nil, fmt.Errorf("failed to initialize Gossipsub: %w", err)
	}

	n.host.SetStreamHandler("/network/1.0.0/request", n.handleRequest)

	// Set network notifications
	host.Network().Notify(n)

	return n, nil
}

// Run run the network
func (n *Network) Run(ctx context.Context) error {
	n.startMDNSDiscovery()

	slog.With("module", "network").Info("network started", "addresses", n.GetLocalAddresses())

	// Wait for context cancellation
	<-ctx.Done()

	// Stop mDNS service (if started)
	if n.mdnsService != nil {
		n.mdnsService.Close()
	}

	// Close host
	if n.host != nil {
		n.host.Close()
	}

	return ctx.Err()
}

// initializeGossipsub initialize Gossipsub
func (n *Network) initializeGossipsub() error {
	var opts []pubsub.Option

	// Enable message signing to ensure message source verifiability
	opts = append(opts, pubsub.WithMessageSignaturePolicy(pubsub.StrictSign))
	opts = append(opts, pubsub.WithMessageIdFn(func(pmsg *pubsub_pb.Message) string {
		// Generate unique message ID using message data and sender ID
		h := sha256.Sum256(pmsg.Data)
		return string(h[:])
	}))

	// Configure Peer scoring system if Peer scoring is enabled
	if n.config.EnablePeerScoring {
		// Set Peer scoring parameters to control publishing permissions
		scoreParams := &pubsub.PeerScoreParams{
			Topics: map[string]*pubsub.TopicScoreParams{},
			AppSpecificScore: func(p peer.ID) float64 {
				return n.config.AppSpecificScore
			},
			AppSpecificWeight:           1.0,
			IPColocationFactorWeight:    n.config.IPColocationWeight,
			IPColocationFactorThreshold: n.config.MaxIPColocation,
			BehaviourPenaltyWeight:      n.config.BehaviourWeight,
			BehaviourPenaltyDecay:       n.config.BehaviourDecay,
			DecayInterval:               time.Hour,
			DecayToZero:                 0.01,
			RetainScore:                 24 * time.Hour,
		}

		// Set Peer scoring thresholds
		thresholds := &pubsub.PeerScoreThresholds{
			GossipThreshold:             -100, // Suppress gossip propagation when below this threshold
			PublishThreshold:            -200, // Should not publish messages when below this threshold (must <= GossipThreshold)
			GraylistThreshold:           -500, // Completely suppress message processing when below this threshold (must <= PublishThreshold)
			AcceptPXThreshold:           1000, // Accept PX when above this threshold
			OpportunisticGraftThreshold: 50,   // Grid score median threshold that triggers opportunistic graft
		}

		opts = append(opts, pubsub.WithPeerScore(scoreParams, thresholds))

	}

	// Create Gossipsub instance
	pubsubInstance, err := pubsub.NewGossipSub(n.ctx, n.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Gossipsub: %w", err)
	}

	n.pubsub = pubsubInstance
	return nil
}

// GetPeers get connected nodes
func (n *Network) GetPeers() []string {
	peers := n.host.Network().Peers()
	result := make([]string, len(peers))
	for i, p := range peers {
		result[i] = p.String()
	}

	return result
}

// ConnectToPeer connect to specified node (external API)
func (n *Network) ConnectToPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	if err := n.host.Connect(n.ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// GetLocalAddresses get local node address list
func (n *Network) GetLocalAddresses() []string {
	if n.host == nil {
		return []string{}
	}

	addresses := n.host.Addrs()
	result := make([]string, len(addresses))

	for i, addr := range addresses {
		// Combine address and node ID into complete p2p address
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), n.host.ID().String())
		result[i] = fullAddr
	}

	return result
}

// BroadcastMessage broadcast message
func (n *Network) BroadcastMessage(topic string, data []byte) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topicObj, err := n.getOrCreateTopic(topic)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	if err := topicObj.Publish(n.ctx, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// getOrCreateTopic get or create topic
func (n *Network) getOrCreateTopic(topicName string) (*pubsub.Topic, error) {
	n.topicsMu.Lock()
	defer n.topicsMu.Unlock()

	if topic, exists := n.topics[topicName]; exists {
		return topic, nil
	}

	topic, err := n.pubsub.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	// Automatically register topic validator
	if err := n.autoRegisterTopicValidator(topicName); err != nil {
		// Log but don't interrupt the process
		slog.With("module", "network.pubsub").Warn("failed to register topic validator", "topic", topicName, "error", err)
	}

	n.topics[topicName] = topic
	return topic, nil
}

// subscribeToTopicInternal internal method to subscribe to topic
func (n *Network) subscribeToTopicInternal(topicName string) error {
	if n.pubsub == nil {
		return fmt.Errorf("gossipsub not initialized")
	}

	topic, err := n.getOrCreateTopic(topicName)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	go n.handleSubscription(topicName, subscription)
	return nil
}

// handleSubscription handle subscription
func (n *Network) handleSubscription(topicName string, subscription *pubsub.Subscription) {
	defer subscription.Cancel()

	for {
		msg, err := subscription.Next(n.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			slog.With("module", "network.pubsub").Error("failed to receive message", "error", err)
			continue
		}

		if msg.ReceivedFrom == n.host.ID() {
			continue // Ignore messages sent by yourself
		}

		go n.processPubsubMessage(topicName, msg)
	}
}

// processPubsubMessage process pubsub message
func (n *Network) processPubsubMessage(topicName string, msg *pubsub.Message) {
	// Apply filters
	n.filtersMu.RLock()

	// Check if there is a general filter
	filter, filterExists := n.messageFilters[topicName]

	n.filtersMu.RUnlock()

	// Apply general filter
	if filterExists {
		// Discard message if filter returns false
		if !filter(msg.ReceivedFrom.String(), topicName, msg.Data) {
			return
		}
	}

	// Call message handler
	n.handlersMu.RLock()
	handler, exists := n.messageHandlers[topicName]
	n.handlersMu.RUnlock()

	if !exists {
		return
	}

	if err := handler(msg.ReceivedFrom.String(), topicName, msg.Data); err != nil {
		slog.With("module", "network.pubsub").Error("failed to process message", "error", err)
	}
}

// RegisterMessageHandler register message handler
// Automatically subscribe to the corresponding topic to receive messages
func (n *Network) RegisterMessageHandler(topic string, handler MessageHandler) {
	n.handlersMu.Lock()
	defer n.handlersMu.Unlock()

	// Check if a handler for this topic has already been registered
	_, exists := n.messageHandlers[topic]
	n.messageHandlers[topic] = handler

	// Automatically subscribe if this is a newly registered topic
	if !exists {
		go func() {
			if err := n.subscribeToTopicInternal(topic); err != nil {
				slog.With("module", "network.pubsub").Warn("failed to auto-subscribe to topic", "topic", topic, "error", err)
			} else {
				slog.With("module", "network.pubsub").Info("auto-subscribed to topic", "topic", topic)
			}
		}()
	}
}

// RegisterMessageFilter register message filter
func (n *Network) RegisterMessageFilter(topic string, filter MessageFilter) {
	n.filtersMu.Lock()
	defer n.filtersMu.Unlock()
	n.messageFilters[topic] = filter
}

// handleRequest handle point-to-point request
func (n *Network) handleRequest(stream libp2pnetwork.Stream) {
	// Use defer to ensure the stream is properly closed when the function exits
	defer stream.Close()

	slog.With("module", "network.request").Info("handling request from peer", "peer", stream.Conn().RemotePeer().String())

	// Set read timeout
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		slog.With("module", "network.request").Error("failed to set read deadline", "error", err)
		return
	}

	// Use more efficient IO reading method
	data, err := io.ReadAll(stream)
	if err != nil {
		slog.With("module", "network.request").Error("failed to read request data", "error", err)
		return
	}

	// Reset timeout settings, prepare to write response
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		slog.With("module", "network.request").Error("failed to set write deadline", "error", err)
		return
	}

	if len(data) == 0 {
		slog.With("module", "network.request").Warn("received empty request data")
		return
	}

	// Parse request
	var req netRequest
	if err := req.Deserialize(data); err != nil {
		slog.With("module", "network.request").Error("failed to deserialize request", "error", err)
		// Send error response
		resp := netResponse{
			Type: "error",
			Data: []byte("failed to deserialize request: " + err.Error()),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	slog.With("module", "network.request").Info("parsed request", "type", req.Type, "data_length", len(req.Data))

	// Get request handler
	n.requestMu.RLock()
	handler, exists := n.requestHandlers[req.Type]
	n.requestMu.RUnlock()

	if !exists {
		slog.With("module", "network.request").Warn("no handler found for request type", "type", req.Type)
		// Send error response
		resp := netResponse{
			Type: "error",
			Data: []byte("no handler found for request type: " + req.Type),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	// Process request
	respData, err := handler(stream.Conn().RemotePeer().String(), req.Type, req.Data)
	if err != nil {
		slog.With("module", "network.request").Error("failed to process request", "error", err)
		// Send error response
		resp := netResponse{
			Type: "error",
			Data: []byte(err.Error()),
		}
		respBytes, _ := resp.Serialize()
		if _, err := stream.Write(respBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send error response", "error", err)
		}
		return
	}

	// Send response
	resp := netResponse{
		Type: req.Type,
		Data: respData,
	}
	respBytes, err := resp.Serialize()
	if err != nil {
		slog.With("module", "network.request").Error("failed to serialize response", "error", err)
		// Send serialization error response
		errorResp := netResponse{
			Type: "error",
			Data: []byte("failed to serialize response: " + err.Error()),
		}
		errorRespBytes, _ := errorResp.Serialize()
		if _, err := stream.Write(errorRespBytes); err != nil {
			slog.With("module", "network.request").Error("failed to send serialization error response", "error", err)
		}
		return
	}

	slog.With("module", "network.request").Info("sending response", "type", req.Type, "response_bytes", len(respBytes))
	if _, err := stream.Write(respBytes); err != nil {
		slog.With("module", "network.request").Error("failed to send response", "error", err)
	} else {
		slog.With("module", "network.request").Info("response sent successfully")
		stream.CloseWrite()
	}
}

// RegisterRequestHandler register point-to-point request handler
func (n *Network) RegisterRequestHandler(requestType string, handler RequestHandler) {
	n.requestMu.Lock()
	defer n.requestMu.Unlock()
	n.requestHandlers[requestType] = handler
	slog.With("module", "network.request").Info("registered request handler for type", "type", requestType)
}

// SendRequest send point-to-point request
func (n *Network) SendRequest(peerID string, requestType string, data []byte) ([]byte, error) {
	pID, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}
	ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
	defer cancel()

	// Create stream
	stream, err := n.host.NewStream(ctx, pID, "/network/1.0.0/request")
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// Set write timeout
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Construct request
	req := netRequest{
		Type: requestType,
		Data: data,
	}

	// Serialize request
	reqData, err := req.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	// Send request
	if _, err := stream.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Ensure request data is sent
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("failed to close write: %w", err)
	}

	// Set read timeout
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Use more efficient IO reading method
	responseData, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.With("module", "network.request").Info("read response", "bytes_read", len(responseData))

	// Check if there is response data
	if len(responseData) == 0 {
		return nil, fmt.Errorf("empty response received")
	}

	// Parse response
	var resp netResponse
	if err := resp.Deserialize(responseData); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	// Check if it is an error response
	if resp.Type == "error" {
		return nil, fmt.Errorf("remote error: %s", string(resp.Data))
	}

	return resp.Data, nil
}

// getOrGeneratePrivateKey Get or generate private key
func getOrGeneratePrivateKey(keyPath string) (crypto.PrivKey, error) {
	// If private key file path is specified
	if keyPath != "" {
		// Check if file exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			// File does not exist, generate new private key and save
			slog.With("module", "network.security").Info("private key file not found, generating new key", "path", keyPath)
			priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %w", err)
			}

			// Save private key to file
			if err := SavePrivateKeyToFile(priv, keyPath); err != nil {
				return nil, fmt.Errorf("failed to save private key: %w", err)
			}

			slog.With("module", "network.security").Info("new private key saved", "path", keyPath)
			return priv, nil
		}

		// File exists, try to read
		priv, err := loadPrivateKeyFromFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		slog.With("module", "network.security").Info("private key loaded from file", "path", keyPath)
		return priv, nil
	}

	// No private key file path specified, generate temporary private key
	slog.With("module", "network.security").Info("no private key path specified, generating temporary key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return priv, nil
}

// loadPrivateKeyFromFile Load private key from file
func loadPrivateKeyFromFile(path string) (crypto.PrivKey, error) {
	// Read private key file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Parse PEM format
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM format private key file")
	}

	// Parse private key
	priv, err := crypto.UnmarshalPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return priv, nil
}

// SavePrivateKeyToFile Save private key to file
func SavePrivateKeyToFile(priv crypto.PrivKey, path string) error {
	// Serialize private key
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to serialize private key: %w", err)
	}

	// Create PEM block
	block := &pem.Block{
		Type:  "LIBP2P PRIVATE KEY",
		Bytes: privBytes,
	}

	// Encode to PEM format
	pemData := pem.EncodeToMemory(block)

	// Write to file
	err = os.WriteFile(path, pemData, 0600) // Owner read/write only
	if err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	return nil
}

// startMDNSDiscovery Start mDNS discovery
func (n *Network) startMDNSDiscovery() {
	// Create mDNS service with service name _network_discovery
	n.mdnsService = mdns.NewMdnsService(n.host, "_network_discovery", n)

	// Start mDNS service
	if err := n.mdnsService.Start(); err != nil {
		slog.With("module", "network.discovery").Error("failed to start mDNS service", "error", err)
		return
	}

	slog.With("module", "network.discovery").Info("mDNS service started with service name: _network_discovery")
}

// HandlePeerFound Implement mdns.Notifee interface to handle discovered peers
func (n *Network) HandlePeerFound(pi peer.AddrInfo) {
	// Avoid connecting to self
	if pi.ID == n.host.ID() {
		return
	}

	slog.With("module", "network.discovery").Info("mDNS peer discovered", "peer_id", pi.ID.String(), "addresses", pi.Addrs)

	if err := n.host.Connect(n.ctx, pi); err != nil {
		slog.With("module", "network.discovery").Warn("failed to connect to discovered peer", "peer_id", pi.ID.String(), "error", err)
	} else {
		slog.With("module", "network.discovery").Info("successfully connected to discovered peer", "peer_id", pi.ID.String())
	}
}

// Implement network.Notifiee interface
func (n *Network) Listen(libp2pnetwork.Network, multiaddr.Multiaddr)      {}
func (n *Network) ListenClose(libp2pnetwork.Network, multiaddr.Multiaddr) {}
func (n *Network) Connected(net libp2pnetwork.Network, conn libp2pnetwork.Conn) {
	peerID := conn.RemotePeer()
	slog.With("module", "network.connection").Info("peer connected", "peer_id", peerID.String())
}
func (n *Network) Disconnected(net libp2pnetwork.Network, conn libp2pnetwork.Conn) {
	peerID := conn.RemotePeer()
	slog.With("module", "network.connection").Info("peer disconnected", "peer_id", peerID.String())
}
func (n *Network) OpenedStream(libp2pnetwork.Network, libp2pnetwork.Stream) {}
func (n *Network) ClosedStream(libp2pnetwork.Network, libp2pnetwork.Stream) {}

// IsPeerConnected Check if peer is connected
func (n *Network) IsPeerConnected(peerID peer.ID) bool {
	conns := n.host.Network().ConnsToPeer(peerID)
	return len(conns) > 0
}

// GetLocalPeerID Get local peer ID
func (n *Network) GetLocalPeerID() string {
	if n.host == nil {
		return ""
	}
	return n.host.ID().String()
}
