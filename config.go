package network

// NetworkConfig network configuration
type NetworkConfig struct {
	Host           string   // Listening address
	Port           int      // Listening port
	MaxPeers       int      // Maximum number of connections
	PrivateKeyPath string   // Private key file path
	BootstrapPeers []string // Bootstrap node address list

	// Newly added configuration fields
	EnablePeerScoring  bool    // Whether to enable Peer scoring
	MaxIPColocation    int     // Maximum number of nodes per IP address
	IPColocationWeight float64 // IP colocation weight
	BehaviourWeight    float64 // Behavior weight
	BehaviourDecay     float64 // Behavior decay factor
	AppSpecificScore   float64 // Application specific score
}

// NewNetworkConfig creates a new network configuration instance with default values
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		Host:     "0.0.0.0",
		Port:     0,
		MaxPeers: 100,

		// Peer scoring enabled by default
		EnablePeerScoring:  true,
		MaxIPColocation:    3,    // Maximum 3 nodes per IP address
		IPColocationWeight: -0.1, // IP colocation weight
		BehaviourWeight:    -1.0, // Behavior weight
		BehaviourDecay:     0.98, // Behavior decay factor
	}
}
