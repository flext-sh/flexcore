#!/bin/bash

# Start Real Distributed FlexCore Cluster
# This script starts a 3-node FlexCore cluster with real Windmill integration

set -e

echo "🌟 Starting Real Distributed FlexCore Cluster..."

# Check if infrastructure is running
echo "🔍 Checking infrastructure..."
if ! docker ps | grep -q windmill-server; then
	echo "⚠️  Windmill server not running. Starting infrastructure..."
	docker-compose -f docker-compose.real-windmill.yml up -d
	echo "⏳ Waiting for Windmill to be ready..."
	sleep 30
fi

# Wait for Windmill to be fully ready
echo "⏳ Waiting for Windmill API to be ready..."
for i in {1..30}; do
	if curl -s http://localhost:8000/api/version >/dev/null 2>&1; then
		echo "✅ Windmill is ready!"
		break
	fi
	if [ "$i" -eq 30 ]; then
		echo "❌ Windmill failed to start within timeout"
		exit 1
	fi
	sleep 2
done

# Export common environment variables
export WINDMILL_URL="http://localhost:8000"
export WINDMILL_TOKEN="your-windmill-token-here"
export POSTGRES_URL="postgresql://flexcore:flexcore123@localhost:5433/flexcore"
export REDIS_URL="redis://localhost:6380"
export PLUGIN_DIR="$(pwd)/runtime-plugins"

echo "🚀 Starting FlexCore cluster nodes..."

# Start Node 1 (Leader Candidate)
echo "📡 Starting Node 1 (Leader Candidate)..."
NODE_ID="node-1" \
	NODE_TYPE="leader-candidate" \
	HTTP_PORT=8001 \
	CLUSTER_SIZE=3 \
	./flexcore-node >logs/node-1.log 2>&1 &
NODE1_PID=$!
echo "✅ Node 1 started (PID: $NODE1_PID)"

# Start Node 2 (Worker)
echo "📡 Starting Node 2 (Worker)..."
NODE_ID="node-2" \
	NODE_TYPE="worker" \
	HTTP_PORT=8002 \
	CLUSTER_SIZE=3 \
	./flexcore-node >logs/node-2.log 2>&1 &
NODE2_PID=$!
echo "✅ Node 2 started (PID: $NODE2_PID)"

# Start Node 3 (Worker)
echo "📡 Starting Node 3 (Worker)..."
NODE_ID="node-3" \
	NODE_TYPE="worker" \
	HTTP_PORT=8003 \
	CLUSTER_SIZE=3 \
	./flexcore-node >logs/node-3.log 2>&1 &
NODE3_PID=$!
echo "✅ Node 3 started (PID: $NODE3_PID)"

# Create logs directory if it doesn't exist
mkdir -p logs

# Save PIDs for shutdown
echo "$NODE1_PID" >.node1.pid
echo "$NODE2_PID" >.node2.pid
echo "$NODE3_PID" >.node3.pid

echo ""
echo "🎉 Real Distributed FlexCore Cluster Started!"
echo ""
echo "📊 Cluster Status:"
echo "  🎯 Node 1 (Leader): http://localhost:8001/health"
echo "  ⚡ Node 2 (Worker): http://localhost:8002/health"
echo "  ⚡ Node 3 (Worker): http://localhost:8003/health"
echo ""
echo "🔗 Real Integrations:"
echo "  🌊 Windmill Server: http://localhost:8000"
echo "  💾 PostgreSQL: localhost:5433"
echo "  ⚡ Redis: localhost:6380"
echo ""
echo "📋 Management Commands:"
echo "  📊 Check status: ./check-cluster-status.sh"
echo "  🧪 Run tests: ./test-real-distributed.sh"
echo "  ⏹️  Stop cluster: ./stop-cluster.sh"
echo ""
echo "📝 Logs:"
echo "  👁️  tail -f logs/node-1.log"
echo "  👁️  tail -f logs/node-2.log"
echo "  👁️  tail -f logs/node-3.log"

# Wait for nodes to be ready
echo "⏳ Waiting for cluster to be ready..."
sleep 10

echo "🔍 Testing cluster connectivity..."
for port in 8001 8002 8003; do
	if curl -s http://localhost:$port/health >/dev/null; then
		echo "✅ Node on port $port is responding"
	else
		echo "⚠️  Node on port $port not yet ready"
	fi
done

echo ""
echo "🚀 Cluster is running! Check logs and run tests to verify functionality."
