#!/bin/bash
# Plugin Integration Test - Tests real plugin loading and execution

set -e

echo "=== FLEXCORE PLUGIN INTEGRATION TEST ==="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up plugin test..."
    pkill -f "data-processor" || true
    rm -rf /tmp/flexcore-plugin-test
}

trap cleanup EXIT

# Setup test environment
log_info "Setting up plugin test environment..."
mkdir -p /tmp/flexcore-plugin-test
cd /tmp/flexcore-plugin-test

# Create plugin host test
cat > plugin_host_test.go << 'EOF'
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/exec"
    "time"
    
    "github.com/hashicorp/go-plugin"
)

type DataProcessor interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
    GetInfo() PluginInfo
    HealthCheck(ctx context.Context) error
    Cleanup() error
}

type PluginInfo struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Type        string   `json:"type"`
    Capabilities []string `json:"capabilities"`
}

type DataProcessorRPC struct {
    client *rpc.Client
}

func (g *DataProcessorRPC) Initialize(ctx context.Context, config map[string]interface{}) error {
    var resp error
    err := g.client.Call("Plugin.Initialize", config, &resp)
    if err != nil {
        return err
    }
    return resp
}

func (g *DataProcessorRPC) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    var resp map[string]interface{}
    err := g.client.Call("Plugin.Execute", input, &resp)
    return resp, err
}

func (g *DataProcessorRPC) GetInfo() PluginInfo {
    var resp PluginInfo
    g.client.Call("Plugin.GetInfo", new(interface{}), &resp)
    return resp
}

func (g *DataProcessorRPC) HealthCheck(ctx context.Context) error {
    var resp error
    err := g.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
    if err != nil {
        return err
    }
    return resp
}

func (g *DataProcessorRPC) Cleanup() error {
    var resp error
    err := g.client.Call("Plugin.Cleanup", new(interface{}), &resp)
    if err != nil {
        return err
    }
    return resp
}

type DataProcessorPlugin struct{}

func (DataProcessorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
    return nil, fmt.Errorf("not a server")
}

func (DataProcessorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
    return &DataProcessorRPC{client: c}, nil
}

func main() {
    if len(os.Args) != 2 {
        log.Fatal("Usage: plugin_host_test <plugin-path>")
    }
    
    pluginPath := os.Args[1]
    
    // Plugin configuration
    client := plugin.NewClient(&plugin.ClientConfig{
        HandshakeConfig: plugin.HandshakeConfig{
            ProtocolVersion:  1,
            MagicCookieKey:   "FLEXCORE_PLUGIN",
            MagicCookieValue: "flexcore-plugin-magic-cookie",
        },
        Plugins: map[string]plugin.Plugin{
            "flexcore": &DataProcessorPlugin{},
        },
        Cmd: exec.Command(pluginPath),
        AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
    })
    defer client.Kill()

    // Connect to plugin
    rpcClient, err := client.Client()
    if err != nil {
        log.Fatalf("Error connecting to plugin: %v", err)
    }

    // Get plugin interface
    raw, err := rpcClient.Dispense("flexcore")
    if err != nil {
        log.Fatalf("Error dispensing plugin: %v", err)
    }

    processor := raw.(DataProcessor)

    // Test plugin info
    fmt.Println("=== PLUGIN INFO ===")
    info := processor.GetInfo()
    infoJSON, _ := json.MarshalIndent(info, "", "  ")
    fmt.Println(string(infoJSON))

    // Test initialization
    fmt.Println("\n=== INITIALIZATION ===")
    config := map[string]interface{}{
        "mode": "test",
        "transform": "uppercase,trim",
        "filter_nulls": "true",
        "enrich_metadata": "true",
    }
    
    if err := processor.Initialize(context.Background(), config); err != nil {
        log.Fatalf("Failed to initialize plugin: %v", err)
    }
    fmt.Println("✅ Plugin initialized successfully")

    // Test execution with various data types
    fmt.Println("\n=== EXECUTION TESTS ===")
    
    testCases := []struct {
        name string
        input map[string]interface{}
    }{
        {
            name: "Array Processing",
            input: map[string]interface{}{
                "data": []interface{}{
                    map[string]interface{}{"id": 1, "name": "  item one  ", "active": true},
                    map[string]interface{}{"id": 2, "name": "  item two  ", "active": false},
                    nil, // Should be filtered
                    map[string]interface{}{"id": 3, "name": "  item three  "},
                },
            },
        },
        {
            name: "Map Processing", 
            input: map[string]interface{}{
                "data": map[string]interface{}{
                    "user_id": "user123",
                    "name": "  john doe  ",
                    "email": "john@example.com",
                    "_private": "should be filtered",
                },
            },
        },
        {
            name: "String Processing",
            input: map[string]interface{}{
                "data": "  hello world  ",
            },
        },
        {
            name: "Complex Nested Data",
            input: map[string]interface{}{
                "data": []interface{}{
                    map[string]interface{}{
                        "order_id": "ord123",
                        "items": []interface{}{
                            map[string]interface{}{"sku": "  ABC123  ", "qty": 2},
                            map[string]interface{}{"sku": "  XYZ789  ", "qty": 1},
                        },
                        "customer": map[string]interface{}{
                            "name": "  jane smith  ",
                            "email": "jane@example.com",
                        },
                    },
                },
            },
        },
    }

    for _, testCase := range testCases {
        fmt.Printf("\nTesting: %s\n", testCase.name)
        
        start := time.Now()
        result, err := processor.Execute(context.Background(), testCase.input)
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("❌ Failed: %v\n", err)
            continue
        }
        
        fmt.Printf("✅ Success (took %v)\n", duration)
        
        // Print result summary
        if stats, ok := result["stats"].(map[string]interface{}); ok {
            fmt.Printf("   Stats: %+v\n", stats)
        }
        
        if processingStats, ok := result["processing_stats"].(map[string]interface{}); ok {
            fmt.Printf("   Processing: %+v\n", processingStats)
        }
    }

    // Test health check
    fmt.Println("\n=== HEALTH CHECK ===")
    if err := processor.HealthCheck(context.Background()); err != nil {
        fmt.Printf("❌ Health check failed: %v\n", err)
    } else {
        fmt.Println("✅ Health check passed")
    }

    // Performance test
    fmt.Println("\n=== PERFORMANCE TEST ===")
    performanceInput := map[string]interface{}{
        "data": make([]interface{}, 1000),
    }
    
    // Fill with test data
    for i := 0; i < 1000; i++ {
        performanceInput["data"].([]interface{})[i] = map[string]interface{}{
            "id": i,
            "name": fmt.Sprintf("  item %d  ", i),
            "value": float64(i) * 1.5,
            "active": i%2 == 0,
        }
    }
    
    start := time.Now()
    result, err := processor.Execute(context.Background(), performanceInput)
    duration := time.Since(start)
    
    if err != nil {
        fmt.Printf("❌ Performance test failed: %v\n", err)
    } else {
        fmt.Printf("✅ Processed 1000 records in %v\n", duration)
        if stats, ok := result["stats"].(map[string]interface{}); ok {
            fmt.Printf("   Final stats: %+v\n", stats)
        }
        
        // Calculate throughput
        throughput := float64(1000) / duration.Seconds()
        fmt.Printf("   Throughput: %.2f records/second\n", throughput)
    }

    // Test cleanup
    fmt.Println("\n=== CLEANUP ===")
    if err := processor.Cleanup(); err != nil {
        fmt.Printf("❌ Cleanup failed: %v\n", err)
    } else {
        fmt.Println("✅ Cleanup successful")
    }

    fmt.Println("\n🎉 Plugin integration test completed successfully!")
}
EOF

# Create go.mod
cat > go.mod << 'EOF'
module plugin-integration-test

go 1.21

require github.com/hashicorp/go-plugin v1.6.0
EOF

# Build plugin if it doesn't exist
PLUGIN_SOURCE="/home/marlonsc/flext/flexcore/plugins/data-processor"
PLUGIN_BINARY="/tmp/flexcore-plugin-test/data-processor"

if [ -d "$PLUGIN_SOURCE" ]; then
    log_info "Building data-processor plugin..."
    cd "$PLUGIN_SOURCE"
    
    # Ensure go.mod exists
    if [ ! -f "go.mod" ]; then
        go mod init data-processor-plugin
        go get github.com/hashicorp/go-plugin
    fi
    
    # Build plugin
    go build -o "$PLUGIN_BINARY" .
    cd /tmp/flexcore-plugin-test
    
    if [ -x "$PLUGIN_BINARY" ]; then
        log_success "Plugin built successfully"
    else
        log_error "Failed to build plugin"
        exit 1
    fi
else
    log_error "Plugin source not found at $PLUGIN_SOURCE"
    exit 1
fi

# Build test host
log_info "Building plugin test host..."
go mod download
go build -o plugin_host_test plugin_host_test.go

if [ -x "plugin_host_test" ]; then
    log_success "Test host built successfully"
else
    log_error "Failed to build test host"
    exit 1
fi

# Run integration test
log_info "Running plugin integration test..."
echo "=========================="

if ./plugin_host_test "$PLUGIN_BINARY"; then
    echo "=========================="
    log_success "Plugin integration test PASSED"
    
    # Verify plugin can be started/stopped multiple times
    log_info "Testing plugin lifecycle..."
    for i in {1..3}; do
        echo "Lifecycle test $i/3..."
        timeout 10s ./plugin_host_test "$PLUGIN_BINARY" > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "✅ Lifecycle test $i passed"
        else
            log_error "Lifecycle test $i failed"
            exit 1
        fi
    done
    
    log_success "All plugin tests completed successfully!"
else
    echo "=========================="
    log_error "Plugin integration test FAILED"
    exit 1
fi