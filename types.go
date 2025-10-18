package network

import (
	"context"
	"fmt"
)

// NetMessage Message structure in broadcast mode
type NetMessage struct {
	From  string // Peer ID of the message sender
	Topic string // Topic to which the message belongs
	Data  []byte // Message data
}

// Request Point-to-point request structure
type Request struct {
	Type string // Request type
	Data []byte // Request data
}

// Serialize Serialize request
func (r *Request) Serialize() ([]byte, error) {
	// Simple serialization implementation, in actual projects more complex serialization methods such as protobuf can be used
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize Deserialize request
func (r *Request) Deserialize(data []byte) error {
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

// Response Point-to-point response structure
type Response struct {
	Type string // Response type
	Data []byte // Response data
}

// Serialize Serialize response
func (r *Response) Serialize() ([]byte, error) {
	// Simple serialization implementation, in actual projects more complex serialization methods such as protobuf can be used
	data := make([]byte, 0)
	data = append(data, byte(len(r.Type)))
	data = append(data, []byte(r.Type)...)
	data = append(data, r.Data...)
	return data, nil
}

// Deserialize Deserialize response
func (r *Response) Deserialize(data []byte) error {
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
type MessageHandler func(from string, msg NetMessage) error

// RequestHandler Used to handle requests in point-to-point mode
type RequestHandler func(from string, req Request) ([]byte, error)

// MessageFilter Used to filter broadcast messages
type MessageFilter func(msg NetMessage) bool

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
