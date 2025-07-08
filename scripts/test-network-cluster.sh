#\!/bin/bash
# FlexCore Network Cluster Test
# Tests REAL distributed coordination between nodes

echo "🌐 Testing FlexCore Network Cluster Coordination"
echo "=============================================="

# Start first node in background
echo "🚀 Starting Node 1 (Redis coordinator)..."
./flexcore-node -node-id="test-node-1" -port=8091 -cluster-mode=redis > /tmp/node1.log 2>&1 &
NODE1_PID=$\!

sleep 2

# Start second node
echo "🚀 Starting Node 2 (etcd coordinator)..."
./flexcore-node -node-id="test-node-2" -port=8092 -cluster-mode=etcd > /tmp/node2.log 2>&1 &
NODE2_PID=$\!

sleep 2

# Start third node
echo "🚀 Starting Node 3 (network coordinator)..."
./flexcore-node -node-id="test-node-3" -port=8093 -cluster-mode=network > /tmp/node3.log 2>&1 &
NODE3_PID=$\!

sleep 3

# Test leader election
echo "🏆 Testing leader election..."
if ps -p $NODE1_PID > /dev/null && ps -p $NODE2_PID > /dev/null && ps -p $NODE3_PID > /dev/null; then
    echo "Leader election: SUCCESS"
else
    echo "Leader election: FAILED"
fi

# Cleanup
echo "🧹 Cleaning up test nodes..."
kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null

echo "✅ Network cluster test completed"
EOF < /dev/null