package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestMessageFilterBasic Basic message filtering test
func TestMessageFilterBasic(t *testing.T) {
	// Create two network instances Node1 and Node2, using random ports
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// Start the running context of both network instances
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	// Run both networks in goroutines
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	// Wait for networks to start
	time.Sleep(500 * time.Millisecond)

	// Register message handler and message filter on Node2, filter rejects messages containing "filtered" keyword
	collector := &messageCollector{}

	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		// Reject messages containing "filtered" keyword
		return !bytes.Contains(msg.Data, []byte("filtered"))
	})

	// Wait for subscriptions to establish
	time.Sleep(1 * time.Second)

	// Broadcast two messages from Node1: one containing "filtered" keyword, the other not containing it
	filteredMessage := []byte("this message should be filtered")
	normalMessage := []byte("this message should be received")

	err := n1.BroadcastMessage("test-topic", filteredMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast filtered message: %v", err)
	}

	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	// Verify Node2 only processes messages that do not contain "filtered" keyword
	messages := collector.getMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d messages", len(messages))
	}

	// Verify the number of messages Node2 receives through the message handler is correct
	if len(messages) > 0 {
		receivedMessage := messages[0]
		if !bytes.Equal(receivedMessage.Data, normalMessage) {
			t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(receivedMessage.Data))
		}
	}

	// Verify filtered messages are not processed
	for _, msg := range messages {
		if bytes.Contains(msg.Data, []byte("filtered")) {
			t.Error("Filtered message was incorrectly processed")
		}
	}

	// Clean up resources
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterNoForward Test that filtered messages are not forwarded
func TestMessageFilterNoForward(t *testing.T) {
	// Create three network instances Node1, Node2 and Node3, using random ports
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)
	n3 := createTestNetwork(t, "127.0.0.1", 0)

	// Start the running context of all three network instances
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	defer cancel3()

	// Run all three networks in goroutines
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n3.Run(ctx3)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 3 run failed with error: %v", err)
		}
	}()

	// Wait for networks to start
	time.Sleep(500 * time.Millisecond)

	// Register filter on Node2 that rejects messages containing "do-not-forward" keyword
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		t.Logf("Filtered message: %t", !bytes.Contains(msg.Data, []byte("do-not-forward")))
		// Reject messages containing "do-not-forward" keyword
		return !bytes.Contains(msg.Data, []byte("do-not-forward"))
	})

	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		t.Logf("Received message: %s", string(msg.Data))
		collector2.addMessage(msg)
		return nil
	})

	// Register message handler on Node3
	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		t.Logf("Received message: %s", string(msg.Data))
		collector3.addMessage(msg)
		return nil
	})

	// Wait for subscriptions to establish
	time.Sleep(1 * time.Second)

	// Broadcast message containing "do-not-forward" keyword from Node1
	filteredMessage := []byte("this message should not be forwarded: do-not-forward")
	err := n1.BroadcastMessage("test-topic", filteredMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast filtered message: %v", err)
	}

	// Wait long enough and verify Node3 does not receive the message
	time.Sleep(2 * time.Second)

	messages3 := collector3.getMessages()
	if len(messages3) > 0 {
		t.Error("Node3 received message that should have been filtered by Node2")
	}

	// Broadcast message that does not contain "do-not-forward" keyword from Node1
	normalMessage := []byte("this message should be forwarded normally")
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	messages2 := collector2.getMessages()
	if len(messages2) == 0 {
		t.Error("Node2 did not receive the normal message")
	}

	// Verify Node3 successfully receives the message
	messages3 = collector3.getMessages()
	if len(messages3) == 0 {
		t.Error("Node3 did not receive the normal message")
	}

	// Verify Node3 receives the message that should have been forwarded by Node2
	for _, msg := range messages3 {
		if !bytes.Equal(msg.Data, normalMessage) {
			t.Errorf("Node3 received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(msg.Data))
		}
	}

	// Clean up resources
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}

// TestMessageFilterDynamic Test dynamic filter rules
func TestMessageFilterDynamic(t *testing.T) {
	// Create two network instances Node1 and Node2, using random ports
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)

	// Start the running context of both network instances
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	// Run both networks in goroutines
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	// Wait for networks to start
	time.Sleep(500 * time.Millisecond)

	// Register message handler and initial filter on Node2
	collector := &messageCollector{}

	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector.addMessage(msg)
		return nil
	})

	// Initial filter: reject messages containing "blocked" keyword
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("blocked"))
	})

	// Wait for subscriptions to establish
	time.Sleep(1 * time.Second)

	// Broadcast message containing "blocked" keyword from Node1
	blockedMessage := []byte("this message is blocked")
	err := n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast blocked message: %v", err)
	}

	// Broadcast normal message from Node1
	normalMessage := []byte("this is a normal message")
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	// Verify initial filter is effective: only normal message is received
	messages := collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message with initial filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, normalMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(normalMessage), string(messages[0].Data))
	}

	// Clear message collector
	collector.clear()

	// Update filter: now reject messages containing "forbidden" keyword
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("forbidden"))
	})

	// Wait for filter update to take effect
	time.Sleep(500 * time.Millisecond)

	// Broadcast message containing "forbidden" keyword from Node1
	forbiddenMessage := []byte("this message is forbidden")
	err = n1.BroadcastMessage("test-topic", forbiddenMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast forbidden message: %v", err)
	}

	// Broadcast normal message from Node1
	anotherNormalMessage := []byte("another normal message")
	err = n1.BroadcastMessage("test-topic", anotherNormalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast another normal message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	// Verify updated filter is effective: only normal message is received
	messages = collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message with updated filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, anotherNormalMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(anotherNormalMessage), string(messages[0].Data))
	}

	// Verify previously allowed message is now rejected (blocked message should pass through new filter)
	collector.clear()

	// Resend previously blocked message
	err = n1.BroadcastMessage("test-topic", blockedMessage)
	if err != nil {
		t.Fatalf("Failed to rebroadcast blocked message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	// Verify blocked message now passes through new filter
	messages = collector.getMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message (previously blocked) with updated filter, got %d messages", len(messages))
	}

	if len(messages) > 0 && !bytes.Equal(messages[0].Data, blockedMessage) {
		t.Errorf("Received incorrect message. Expected: %s, Got: %s", string(blockedMessage), string(messages[0].Data))
	}

	// Clean up resources
	cleanupNetworks(context.Background(), cancel1, n1, n2)
}

// TestMessageFilterMultiple Test multiple filters working together
func TestMessageFilterMultiple(t *testing.T) {
	// Create three network instances Node1, Node2 and Node3, using random ports
	n1 := createTestNetwork(t, "127.0.0.1", 0)
	n2 := createTestNetwork(t, "127.0.0.1", 0)
	n3 := createTestNetwork(t, "127.0.0.1", 0)

	// Start the running context of all three network instances
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	defer cancel3()

	// Run all three networks in goroutines
	go func() {
		err := n1.Run(ctx1)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 1 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n2.Run(ctx2)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 2 run failed with error: %v", err)
		}
	}()

	go func() {
		err := n3.Run(ctx3)
		if err != nil && err != context.Canceled {
			t.Errorf("Network 3 run failed with error: %v", err)
		}
	}()

	// Wait for networks to start
	time.Sleep(500 * time.Millisecond)

	// Register filter on Node2: reject messages containing "node2-blocked" keyword
	n2.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("node2-blocked"))
	})

	// Register filter on Node3: reject messages containing "node3-blocked" keyword
	n3.RegisterMessageFilter("test-topic", func(msg network.NetMessage) bool {
		return !bytes.Contains(msg.Data, []byte("node3-blocked"))
	})

	// Register message handler on Node2 and Node3
	collector2 := &messageCollector{}
	n2.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector2.addMessage(msg)
		return nil
	})

	collector3 := &messageCollector{}
	n3.RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
		collector3.addMessage(msg)
		return nil
	})

	// Wait for subscriptions to establish
	time.Sleep(1 * time.Second)

	// Broadcast three types of messages from Node1
	node2BlockedMessage := []byte("message blocked by node2: node2-blocked")
	node3BlockedMessage := []byte("message blocked by node3: node3-blocked")
	normalMessage := []byte("normal message that should reach both nodes")

	// Broadcast message blocked by Node2
	err := n1.BroadcastMessage("test-topic", node2BlockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast node2-blocked message: %v", err)
	}

	// Broadcast message blocked by Node3
	err = n1.BroadcastMessage("test-topic", node3BlockedMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast node3-blocked message: %v", err)
	}

	// Broadcast normal message
	err = n1.BroadcastMessage("test-topic", normalMessage)
	if err != nil {
		t.Fatalf("Failed to broadcast normal message: %v", err)
	}

	// Wait for message delivery
	time.Sleep(2 * time.Second)

	// Verify Node2 only receives normal message and message blocked by Node3 (Node2 will forward message blocked by Node3)
	messages2 := collector2.getMessages()
	expectedMessages2 := 2 // normal message + message blocked by Node3
	if len(messages2) != expectedMessages2 {
		t.Errorf("Node2 expected %d messages, got %d messages", expectedMessages2, len(messages2))
	}

	// Verify Node3 only receives normal message and message blocked by Node2 (Node3 will receive message blocked by Node2)
	messages3 := collector3.getMessages()
	expectedMessages3 := 2 // normal message + message blocked by Node2
	if len(messages3) != expectedMessages3 {
		t.Errorf("Node3 expected %d messages, got %d messages", expectedMessages3, len(messages3))
	}

	// Verify Node2 does not receive message filtered by its own filter
	for _, msg := range messages2 {
		if bytes.Contains(msg.Data, []byte("node2-blocked")) {
			t.Error("Node2 received message that should have been filtered by its own filter")
		}
	}

	// Verify Node3 does not receive message filtered by its own filter
	for _, msg := range messages3 {
		if bytes.Contains(msg.Data, []byte("node3-blocked")) {
			t.Error("Node3 received message that should have been filtered by its own filter")
		}
	}

	// Verify both nodes receive normal message
	foundNormalInNode2 := false
	foundNormalInNode3 := false

	for _, msg := range messages2 {
		if bytes.Equal(msg.Data, normalMessage) {
			foundNormalInNode2 = true
			break
		}
	}

	for _, msg := range messages3 {
		if bytes.Equal(msg.Data, normalMessage) {
			foundNormalInNode3 = true
			break
		}
	}

	if !foundNormalInNode2 {
		t.Error("Node2 did not receive the normal message")
	}

	if !foundNormalInNode3 {
		t.Error("Node3 did not receive the normal message")
	}

	// Clean up resources
	cleanupNetworks(context.Background(), cancel1, n1, n2, n3)
}
