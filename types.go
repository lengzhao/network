package network

import (
	"context"
	"fmt"
)

// netRequest Point-to-point request structure
type netRequest struct {
	Type string // netRequest type
	Data []byte // netRequest data
}

// Serialize Serialize request
func (r *netRequest) Serialize() ([]byte, error) {
	// Simple serialization implementation, in actual projects more complex serialization methods such as protobuf can be used
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize Deserialize request
func (r *netRequest) Deserialize(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	typeLen := int(data[0])
	if len(data) < 1+typeLen {
		return fmt.Errorf("invalid data format")
	}
	r.Type = string(data[1 : 1+typeLen])
	r.Data = data[1+typeLen:]
	return nil
}

// netResponse Point-to-point response structure
type netResponse struct {
	Type string // netResponse type
	Data []byte // netResponse data
}

// Serialize Serialize response
func (r *netResponse) Serialize() ([]byte, error) {
	// Simple serialization implementation, in actual projects more complex serialization methods such as protobuf can be used
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize Deserialize response
func (r *netResponse) Deserialize(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	typeLen := int(data[0])
	if len(data) < 1+typeLen {
		return fmt.Errorf("invalid data format")
	}
	r.Type = string(data[1 : 1+typeLen])
	r.Data = data[1+typeLen:]
	return nil
}

// MessageHandler Used to handle messages received in broadcast mode
type MessageHandler = func(from string, topic string, data []byte) error

// RequestHandler Used to handle requests in point-to-point mode
type RequestHandler = func(from string, reqType string, data []byte) ([]byte, error)

// MessageFilter Used to filter broadcast messages
type MessageFilter = func(from string, topic string, data []byte) bool

// NetworkInterface Network interface, defines all externally provided functions
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
