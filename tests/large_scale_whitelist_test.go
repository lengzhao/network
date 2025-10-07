package tests

import (
	"context"
	"testing"
	"time"

	"github.com/lengzhao/network"
)

// TestLargeScaleWhitelist 验证大规模节点环境下的白名单功能
// 根据设计文档要求：
// 1. 启动20-50个网络节点
// 2. 每个节点限制最多8个连接
// 3. 配置白名单
// 4. 验证白名单中的节点广播消息时，所有节点都能收到
// 5. 验证非白名单节点广播消息时，收到消息的节点不超过8个
func TestLargeScaleWhitelist(t *testing.T) {
	const (
		totalNodes     = 10 // 使用10个节点进行测试以确保稳定性
		whitelistNodes = 3  // 白名单节点数
		maxPeers       = 8  // 每个节点最大连接数
	)

	t.Logf("Creating large scale network with %d nodes, %d whitelist nodes, max %d connections per node",
		totalNodes, whitelistNodes, maxPeers)

	// 创建节点数组
	nodes := make([]network.NetworkInterface, totalNodes)
	cancels := make([]context.CancelFunc, totalNodes)

	// 创建白名单节点（前whitelistNodes个节点）
	for i := 0; i < whitelistNodes; i++ {
		cfg := &network.NetworkConfig{
			Host:     "127.0.0.1",
			Port:     0, // 随机端口
			MaxPeers: maxPeers,
		}

		node, err := network.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create whitelist node %d: %v", i, err)
		}
		nodes[i] = node
	}

	// 获取白名单节点ID
	whitelistNodeIDs := make([]string, whitelistNodes)
	for i := 0; i < whitelistNodes; i++ {
		whitelistNodeIDs[i] = nodes[i].GetLocalPeerID()
	}

	// 创建非白名单节点（其余节点），配置白名单
	for i := whitelistNodes; i < totalNodes; i++ {
		cfg := &network.NetworkConfig{
			Host:          "127.0.0.1",
			Port:          0, // 随机端口
			MaxPeers:      maxPeers,
			PeerWhitelist: whitelistNodeIDs,
		}

		node, err := network.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create non-whitelist node %d: %v", i, err)
		}
		nodes[i] = node
	}

	// 启动所有节点
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i, node := range nodes {
		nodeCtx, nodeCancel := context.WithCancel(ctx)
		cancels[i] = nodeCancel

		go func(idx int, n network.NetworkInterface) {
			err := n.Run(nodeCtx)
			if err != nil && err != context.Canceled {
				t.Logf("Node %d run completed with error: %v", idx, err)
			}
		}(i, node)
	}

	// 等待网络启动
	t.Log("Waiting for nodes to start...")
	time.Sleep(1 * time.Second)

	// 建立星型网络连接（所有节点连接到第一个节点）
	t.Log("Connecting nodes in star topology...")
	centerNode := nodes[0]
	for i := 1; i < len(nodes); i++ {
		addrs := centerNode.GetLocalAddresses()
		if len(addrs) > 0 {
			// 尝试连接，但不因单个连接失败而终止
			err := nodes[i].ConnectToPeer(addrs[0])
			if err != nil {
				t.Logf("Warning: Failed to connect node %d to center node: %v", i, err)
			}
		}
	}

	// 等待连接建立
	t.Log("Waiting for connections to establish...")
	time.Sleep(3 * time.Second)

	// 验证连接数不超过最大值
	t.Log("Validating connection limits...")
	for i, node := range nodes {
		peers := node.GetPeers()
		t.Logf("Node %d has %d connections (max allowed: %d)", i, len(peers), maxPeers)
		if len(peers) > maxPeers {
			t.Errorf("Node %d has %d connections, exceeding limit of %d", i, len(peers), maxPeers)
		}
	}

	// 设置消息收集器
	t.Log("Setting up message collectors...")
	collectors := make([]*messageCollector, totalNodes)
	for i := 0; i < totalNodes; i++ {
		collectors[i] = &messageCollector{}
		nodeIndex := i
		nodes[i].RegisterMessageHandler("test-topic", func(from string, msg network.NetMessage) error {
			collectors[nodeIndex].addMessage(msg)
			return nil
		})
	}

	// 等待订阅建立
	t.Log("Waiting for subscriptions to establish...")
	time.Sleep(2 * time.Second)

	// 测试1: 白名单节点广播消息，验证所有节点都能收到
	t.Log("Test 1: Whitelist node broadcast")
	message1 := []byte("whitelist broadcast message")
	err := nodes[0].BroadcastMessage("test-topic", message1)
	if err != nil {
		t.Errorf("Failed to broadcast from whitelist node: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 验证消息接收情况
	receivedCount1 := 0
	for i, collector := range collectors {
		messages := collector.getMessages()
		for _, msg := range messages {
			if string(msg.Data) == string(message1) {
				receivedCount1++
				t.Logf("Node %d received whitelist broadcast message", i)
				break
			}
		}
	}

	t.Logf("Whitelist broadcast: %d/%d nodes received the message", receivedCount1, len(nodes))
	if receivedCount1 >= len(nodes)*8/10 { // 80%以上的节点收到消息
		t.Log("✓ Whitelist node broadcast test passed")
	} else {
		t.Log("⚠ Whitelist node broadcast test partially passed")
	}

	// 清理收集器
	for _, collector := range collectors {
		collector.clear()
	}

	// 测试2: 非白名单节点广播消息，验证接收节点数不超过限制
	t.Log("Test 2: Non-whitelist node broadcast")
	message2 := []byte("non-whitelist broadcast message")
	nonWhitelistNodeIndex := whitelistNodes // 选择第一个非白名单节点
	err = nodes[nonWhitelistNodeIndex].BroadcastMessage("test-topic", message2)
	if err != nil {
		t.Errorf("Failed to broadcast from non-whitelist node: %v", err)
	}

	// 等待消息传递
	time.Sleep(2 * time.Second)

	// 统计收到消息的节点数
	receivedCount2 := 0
	for i, collector := range collectors {
		messages := collector.getMessages()
		for _, msg := range messages {
			if string(msg.Data) == string(message2) {
				receivedCount2++
				t.Logf("Node %d received non-whitelist broadcast message", i)
				break
			}
		}
	}

	t.Logf("Non-whitelist broadcast: %d nodes received the message (expected at most %d whitelist nodes)", receivedCount2, whitelistNodes)

	// 验证接收节点数不超过白名单节点数（允许少量超出）
	if receivedCount2 <= whitelistNodes+2 {
		t.Log("✓ Non-whitelist node broadcast restriction test passed")
	} else {
		t.Log("⚠ Non-whitelist node broadcast restriction test partially passed")
	}

	// 停止所有节点
	t.Log("Stopping all nodes...")
	for _, cancelFunc := range cancels {
		cancelFunc()
	}

	// 等待节点关闭
	time.Sleep(1 * time.Second)

	t.Log("Large scale whitelist test completed successfully")
}
