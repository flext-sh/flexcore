#!/bin/bash
echo "Starting FlexCore Distributed Cluster..."

# Kill existing processes
pkill -f "api_gateway" 2>/dev/null || true
pkill -f "cqrs_system" 2>/dev/null || true
pkill -f "script_engine" 2>/dev/null || true
pkill -f "data_processor" 2>/dev/null || true
pkill -f "windmill_server" 2>/dev/null || true
pkill -f "metrics_server" 2>/dev/null || true

sleep 2

# Create logs directory
mkdir -p logs

# Start services
if [ -x ./api_gateway ]; then
	echo "Starting API Gateway..."
	./api_gateway >logs/api_gateway.log 2>&1 &
	echo $! >api_gateway.pid
	echo "API Gateway started (port 8100)"
fi

if [ -x ./cqrs_system ]; then
	echo "Starting CQRS System..."
	./cqrs_system >logs/cqrs_system.log 2>&1 &
	echo $! >cqrs_system.pid
	echo "CQRS System started (port 8099)"
fi

if [ -x ./script_engine ]; then
	echo "Starting Script Engine..."
	./script_engine >logs/script_engine.log 2>&1 &
	echo $! >script_engine.pid
	echo "Script Engine started (port 8098)"
fi

if [ -x ./data_processor ]; then
	echo "Starting Data Processor..."
	./data_processor >logs/data_processor.log 2>&1 &
	echo $! >data_processor.pid
	echo "Data Processor started (port 8097)"
fi

if [ -x ./windmill_server ]; then
	echo "Starting Windmill Server..."
	./windmill_server >logs/windmill_server.log 2>&1 &
	echo $! >windmill_server.pid
	echo "Windmill Server started (port 8096)"
fi

if [ -x ./metrics_server ]; then
	echo "Starting Metrics Server..."
	./metrics_server >logs/metrics_server.log 2>&1 &
	echo $! >metrics_server.pid
	echo "Metrics Server started (port 8080)"
fi

sleep 5
echo "Distributed cluster startup completed"
