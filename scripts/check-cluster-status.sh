#!/bin/bash

# Check Real Distributed FlexCore Cluster Status

echo "🔍 FlexCore Cluster Status Check"
echo "================================"

# Check if nodes are running
echo "📊 Node Status:"
for port in 8001 8002 8003; do
    if curl -s http://localhost:$port/health > /dev/null 2>&1; then
        response=$(curl -s http://localhost:$port/info)
        node_id=$(echo "$response" | jq -r ".node_id" 2>/dev/null || echo "unknown")
        node_type=$(echo "$response" | jq -r ".node_type" 2>/dev/null || echo "unknown")
        plugin_count=$(echo "$response" | jq -r ".plugin_count" 2>/dev/null || echo "0")
        echo "  ✅ $node_id ($node_type) - Port $port - $plugin_count plugins"
    else
        echo "  ❌ Node on port $port - Not responding"
    fi
done

echo ""
echo "🔗 Infrastructure Status:"

# Check Windmill
if curl -s http://localhost:8000/api/version > /dev/null 2>&1; then
    echo "  ✅ Windmill Server - http://localhost:8000"
else
    echo "  ❌ Windmill Server - Not accessible"
fi

# Check PostgreSQL
if PGPASSWORD=flexcore123 psql -h localhost -p 5433 -U flexcore -d flexcore -c "SELECT 1;" > /dev/null 2>&1; then
    echo "  ✅ PostgreSQL Database - localhost:5433"
else
    echo "  ❌ PostgreSQL Database - Not accessible"
fi

# Check Redis
if redis-cli -h localhost -p 6380 ping > /dev/null 2>&1; then
    echo "  ✅ Redis Cache - localhost:6380"
else
    echo "  ❌ Redis Cache - Not accessible"
fi

echo ""
echo "🔌 Plugin Status:"
if curl -s http://localhost:8001/plugins > /dev/null 2>&1; then
    plugins_response=$(curl -s http://localhost:8001/plugins)
    plugin_count=$(echo "$plugins_response" | jq -r ".count" 2>/dev/null || echo "0")
    echo "  📊 Total plugins loaded: $plugin_count"
    
    if [ "$plugin_count" -gt 0 ]; then
        echo "$plugins_response" | jq -r '.plugins[] | "    🔌 \(.name) v\(.version) (\(.type))"' 2>/dev/null
    fi
else
    echo "  ❌ Unable to retrieve plugin status"
fi

echo ""
echo "📋 Quick Actions:"
echo "  🧪 Run tests: ./test-real-distributed.sh"
echo "  📝 View logs: tail -f logs/node-*.log"
echo "  ⏹️  Stop cluster: ./stop-cluster.sh"