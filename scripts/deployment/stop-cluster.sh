#!/bin/bash

# Stop Real Distributed FlexCore Cluster

echo "⏹️  Stopping FlexCore Cluster..."

# Stop FlexCore nodes
if [ -f .node1.pid ]; then
    PID=$(cat .node1.pid)
    if kill -0 $PID 2>/dev/null; then
        echo "🛑 Stopping Node 1 (PID: $PID)..."
        kill $PID
    fi
    rm -f .node1.pid
fi

if [ -f .node2.pid ]; then
    PID=$(cat .node2.pid)
    if kill -0 $PID 2>/dev/null; then
        echo "🛑 Stopping Node 2 (PID: $PID)..."
        kill $PID
    fi
    rm -f .node2.pid
fi

if [ -f .node3.pid ]; then
    PID=$(cat .node3.pid)
    if kill -0 $PID 2>/dev/null; then
        echo "🛑 Stopping Node 3 (PID: $PID)..."
        kill $PID
    fi
    rm -f .node3.pid
fi

# Kill any remaining flexcore-node processes
echo "🧹 Cleaning up any remaining processes..."
pkill -f "flexcore-node" 2>/dev/null || true

echo "⏹️  Stopping infrastructure (optional)..."
echo "To stop NATIVE infrastructure: docker-compose -f deployments/docker/development/docker-compose.native-windmill.yml down"

echo "✅ FlexCore cluster stopped successfully"