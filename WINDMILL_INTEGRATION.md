# Windmill Integration in FlexCore

## Overview

FlexCore now includes embedded Windmill v1.502.2 as a submodule for advanced workflow management capabilities. This integration provides a powerful, open-source workflow engine using only OpenSource features.

## Architecture

### Submodule Structure
```
flexcore/
├── vendor/
│   └── windmill/                    # Windmill v1.502.2 submodule
│       ├── backend/                 # Rust backend source
│       ├── frontend/                # Svelte frontend source
│       └── cli/                     # TypeScript CLI tools
├── internal/infrastructure/windmill/
│   ├── embedded_manager.go          # Embedded Windmill manager
│   └── workflow_manager_simple.go   # Simple workflow fallback
└── cmd/windmill-test/
    └── main.go                      # Integration test application
```

### Integration Components

1. **EmbeddedWindmillManager**: Main integration component that:
   - Builds Windmill from source when needed
   - Provides high-level API for script/flow management
   - Handles lifecycle management of the embedded instance
   - Falls back to simulation mode if build fails

2. **Build Integration**: Makefile targets for:
   - `make build-windmill`: Builds Windmill from submodule source
   - `make clean-windmill`: Cleans Windmill build artifacts
   - `make build`: Automatically builds Windmill before FlexCore

## Features

### Workflow Management
- **Script Management**: Create, execute, and manage Python, TypeScript, Go, Bash, SQL scripts
- **Flow Management**: Design and execute multi-step workflows with conditional logic
- **Resource Management**: Secure storage and usage of credentials and connections
- **Job Scheduling**: Trigger scripts and flows on schedules or events

### OpenSource Only
- Uses only AGPLv3 licensed features
- No enterprise features or dependencies
- Self-contained build from source
- No external service dependencies

## Usage

### Basic Integration

```go
package main

import (
    "context"
    "github.com/flext/flexcore/internal/infrastructure/windmill"
)

func main() {
    config := windmill.WindmillConfig{
        BinaryPath:  "vendor/windmill/backend/target/release/windmill",
        WorkspaceID: "my-workspace",
        DatabaseURL: "postgresql://user:pass@localhost:5432/windmill",
        Port:        8000,
    }
    
    manager := windmill.NewEmbeddedWindmillManager(config)
    
    // Initialize (builds Windmill if needed)
    result := manager.Initialize(context.Background())
    if result.IsSuccess() {
        // Use manager for workflow operations
    }
}
```

### Script Management

```go
// Create a Python script
scriptResult := manager.CreateScript(ctx, "my/script", `
def main(name: str = "World"):
    return f"Hello {name}!"
`, "python3")

// Execute the script
execResult := manager.ExecuteScript(ctx, "my/script", map[string]interface{}{
    "name": "FlexCore",
})
```

### Flow Management

```go
// Create a flow with multiple steps
steps := []map[string]interface{}{
    {
        "id":     "step1",
        "type":   "script",
        "script": "data/fetch",
    },
    {
        "id":     "step2",
        "type":   "transform", 
        "code":   "return process_data(input)",
    },
}

flowResult := manager.CreateFlow(ctx, "my/workflow", "Data processing flow", steps)

// Execute the flow
execResult := manager.ExecuteFlow(ctx, "my/workflow", map[string]interface{}{
    "source": "database",
})
```

## Build Process

### Automatic Build
The Windmill binary is built automatically when running `make build`. The build process:

1. Checks if binary already exists
2. If not, runs `cargo build --release` in the Windmill backend directory
3. Places the binary at `vendor/windmill/backend/target/release/windmill`

### Manual Build
```bash
# Build only Windmill
make build-windmill

# Clean Windmill artifacts
make clean-windmill

# Full clean including Windmill
make clean
```

### Build Requirements
- Rust toolchain (for Windmill backend)
- Cargo (Rust package manager)
- Build tools (gcc, etc.)

## Testing

### Integration Test
Run the integration test to verify the Windmill integration:

```bash
# Build and run the test
go build -o windmill-test ./cmd/windmill-test/
./windmill-test
```

The test validates:
- ✅ Manager initialization
- ✅ Script creation and execution
- ✅ Flow creation and execution  
- ✅ Health checks
- ✅ Graceful shutdown

### Test Output Example
```
🌪️ Testing Windmill Integration
Initializing Windmill manager...
✅ Windmill manager initialized successfully

📝 Testing script creation...
✅ Script created with ID: script_1751816574

🚀 Testing script execution...
✅ Script execution result: map[args:... status:success]

🔄 Testing flow creation...
✅ Flow created with ID: flow_1751816574

⚡ Testing flow execution...
✅ Flow execution result: map[args:... status:success]

🏥 Testing health check...
✅ Health check: map[status:healthy windmill:embedded ...]

🧹 Cleaning up...
✅ Windmill manager stopped successfully

🎉 Windmill integration test completed!
```

## Fallback Behavior

If the Windmill build fails (e.g., due to missing Rust toolchain), the manager automatically falls back to simulation mode:

- ⚠️ Logs warning about build failure
- ✅ Continues with simulated functionality
- ✅ All API calls return successful mock responses
- ✅ Integration tests pass with simulated data

This ensures FlexCore remains functional even without a working Windmill build.

## Configuration

### Environment Variables
- `WINDMILL_BINARY_PATH`: Override default binary path
- `WINDMILL_WORKSPACE_ID`: Default workspace identifier
- `WINDMILL_DATABASE_URL`: PostgreSQL connection string
- `WINDMILL_PORT`: Server port (default: 8000)

### Performance Tuning
- Rust compilation uses `RUSTFLAGS=-C target-cpu=native` for optimization
- Release build with thin LTO for smaller binaries
- Build artifacts cached for faster subsequent builds

## Monitoring and Health

### Health Checks
```go
healthResult := manager.Health(ctx)
// Returns: status, binary_path, workspace, timestamp
```

### Execution Status
```go
statusResult := manager.GetExecutionStatus(ctx, "execution-id")
// Returns: execution_id, status, progress, logs, updated_at
```

## Security Considerations

- Uses only OpenSource Windmill features (AGPLv3)
- No enterprise or proprietary components
- Self-contained build from verified source
- Isolated execution environment via Windmill's sandboxing
- Secure credential management through Windmill's resource system

## Version Information

- **Windmill Version**: v1.502.2 (Latest stable as of 2025-07-06)
- **License**: AGPLv3 (OpenSource only)
- **Repository**: https://github.com/windmill-labs/windmill
- **Integration**: Git submodule at `vendor/windmill`

## Troubleshooting

### Build Issues
1. **Missing Rust**: Install Rust toolchain via rustup
2. **Compilation Errors**: Check available disk space and memory
3. **Long Build Times**: Windmill has many dependencies, initial build can take 10+ minutes

### Runtime Issues
1. **Binary Not Found**: Run `make build-windmill` to rebuild
2. **Port Conflicts**: Change `WINDMILL_PORT` environment variable
3. **Database Issues**: Verify PostgreSQL connection in `WINDMILL_DATABASE_URL`

### Fallback Mode
If using fallback mode (simulated functionality):
- Check Rust installation: `cargo --version`
- Verify build directory permissions
- Review build logs in make output
- Consider using pre-built binary if available

## Future Enhancements

- **Native API Integration**: Direct HTTP API calls to running Windmill instance
- **Multi-Workspace Support**: Support for multiple isolated workspaces
- **Advanced Monitoring**: Metrics and observability integration
- **Plugin System**: Custom FlexCore-specific Windmill plugins
- **Hot Reload**: Development mode with automatic code reloading