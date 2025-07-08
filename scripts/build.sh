#!/bin/bash

# FlexCore Plugin Build Script
# Compiles all plugins into executable binaries

set -e

echo "🔧 Building FlexCore Plugins..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGINS_DIR="$PROJECT_ROOT/plugins"
DIST_DIR="$PROJECT_ROOT/dist/plugins"

echo "📂 Project root: $PROJECT_ROOT"
echo "📂 Plugins directory: $PLUGINS_DIR"
echo "📂 Distribution directory: $DIST_DIR"

# Create dist directory
mkdir -p "$DIST_DIR"

# Function to build a plugin
build_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    
    echo -e "\n🔨 Building plugin: ${BLUE}$plugin_name${NC}"
    
    if [ ! -d "$plugin_dir" ]; then
        echo -e "${RED}❌ Plugin directory not found: $plugin_dir${NC}"
        return 1
    fi
    
    if [ ! -f "$plugin_dir/main.go" ]; then
        echo -e "${RED}❌ main.go not found in: $plugin_dir${NC}"
        return 1
    fi
    
    # Change to plugin directory
    cd "$plugin_dir"
    
    # Initialize go module if go.mod doesn't exist
    if [ ! -f "go.mod" ]; then
        echo "📦 Initializing go module for $plugin_name"
        go mod init "github.com/flext/flexcore/plugins/$plugin_name"
        go mod edit -replace github.com/flext/flexcore=../..
    fi
    
    # Download dependencies
    echo "📦 Downloading dependencies for $plugin_name"
    go mod tidy
    
    # Build the plugin
    echo "🔨 Compiling $plugin_name"
    output_binary="$DIST_DIR/$plugin_name"
    
    # Build with proper flags
    CGO_ENABLED=1 go build \
        -ldflags="-s -w -X main.version=1.0.0 -X main.buildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
        -tags netgo \
        -o "$output_binary" \
        .
    
    # Make executable
    chmod +x "$output_binary"
    
    # Verify the binary
    if [ -f "$output_binary" ]; then
        file_info=$(file "$output_binary")
        size=$(du -h "$output_binary" | cut -f1)
        echo -e "${GREEN}✅ Successfully built: $plugin_name${NC}"
        echo -e "   📁 Location: $output_binary"
        echo -e "   📊 Size: $size"
        echo -e "   🔍 Type: $file_info"
        
        # Test if binary can start (basic smoke test)
        echo "🧪 Testing plugin startup..."
        timeout 2s "$output_binary" || true
        echo -e "${GREEN}✅ Plugin startup test completed${NC}"
    else
        echo -e "${RED}❌ Failed to build: $plugin_name${NC}"
        return 1
    fi
    
    # Return to project root
    cd "$PROJECT_ROOT"
}

# Function to create plugin manifest
create_manifest() {
    local manifest_file="$DIST_DIR/plugins.json"
    echo "📋 Creating plugin manifest: $manifest_file"
    
    cat > "$manifest_file" << EOF
{
    "version": "1.0.0",
    "build_time": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
    "plugins": [
        {
            "name": "postgres-extractor",
            "type": "extractor",
            "version": "1.0.0",
            "binary": "postgres-extractor",
            "description": "PostgreSQL data extraction plugin",
            "capabilities": ["sql-queries", "batch-extraction", "schema-detection", "incremental-sync"],
            "config_schema": {
                "host": {"type": "string", "required": true},
                "port": {"type": "string", "default": "5432"},
                "database": {"type": "string", "required": true},
                "username": {"type": "string", "required": true},
                "password": {"type": "string", "required": true},
                "sslmode": {"type": "string", "default": "disable"}
            }
        },
        {
            "name": "json-transformer",
            "type": "transformer",
            "version": "1.0.0",
            "binary": "json-transformer",
            "description": "JSON data transformation plugin",
            "capabilities": ["clean-data", "normalize-fields", "validate-schema", "type-conversion", "field-mapping"],
            "config_schema": {
                "transformations": {"type": "array", "required": true},
                "required_fields": {"type": "array", "default": []},
                "field_mapping": {"type": "object", "default": {}},
                "filter": {"type": "object", "default": {}}
            }
        },
        {
            "name": "api-loader",
            "type": "loader",
            "version": "1.0.0",
            "binary": "api-loader",
            "description": "HTTP/REST API data loading plugin",
            "capabilities": ["rest-api", "json-payload", "batch-loading", "retry-logic", "custom-headers"],
            "config_schema": {
                "endpoint": {"type": "string", "required": true},
                "method": {"type": "string", "default": "POST"},
                "timeout_seconds": {"type": "number", "default": 30},
                "retries": {"type": "number", "default": 3},
                "batch_size": {"type": "number", "default": 100},
                "headers": {"type": "object", "default": {}}
            }
        }
    ]
}
EOF
    
    echo -e "${GREEN}✅ Plugin manifest created${NC}"
}

# Function to create installation script
create_install_script() {
    local install_script="$DIST_DIR/install.sh"
    echo "📦 Creating installation script: $install_script"
    
    cat > "$install_script" << 'EOF'
#!/bin/bash

# FlexCore Plugin Installation Script

set -e

echo "🚀 Installing FlexCore Plugins..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "🖥️  Detected: $OS/$ARCH"

# Default installation directory
INSTALL_DIR="${FLEXCORE_PLUGIN_DIR:-$HOME/.flexcore/plugins}"

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Copy plugins
echo "📂 Installing plugins to: $INSTALL_DIR"

for plugin in postgres-extractor json-transformer api-loader; do
    if [ -f "./$plugin" ]; then
        cp "./$plugin" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$plugin"
        echo "✅ Installed: $plugin"
    else
        echo "⚠️  Plugin not found: $plugin"
    fi
done

# Copy manifest
if [ -f "./plugins.json" ]; then
    cp "./plugins.json" "$INSTALL_DIR/"
    echo "✅ Installed: plugins.json"
fi

echo ""
echo "🎉 Installation completed!"
echo "📂 Plugins installed in: $INSTALL_DIR"
echo ""
echo "To use plugins, set environment variable:"
echo "export FLEXCORE_PLUGIN_DIR=$INSTALL_DIR"
echo ""
echo "Or add to your FlexCore configuration:"
echo "\"plugin_directory\": \"$INSTALL_DIR\""
EOF
    
    chmod +x "$install_script"
    echo -e "${GREEN}✅ Installation script created${NC}"
}

# Main build process
echo -e "\n🏗️  Starting plugin build process..."

# List of plugins to build
PLUGINS=(
    "postgres-extractor"
    "json-transformer"
    "api-loader"
)

# Build each plugin
success_count=0
total_count=${#PLUGINS[@]}

for plugin in "${PLUGINS[@]}"; do
    if build_plugin "$plugin"; then
        ((success_count++))
    else
        echo -e "${RED}❌ Failed to build $plugin${NC}"
    fi
done

# Create manifest and installation script
create_manifest
create_install_script

# Create README for distribution
cat > "$DIST_DIR/README.md" << EOF
# FlexCore Plugins

This directory contains compiled FlexCore plugins.

## Plugins Included

- **postgres-extractor**: PostgreSQL data extraction
- **json-transformer**: JSON data transformation  
- **api-loader**: HTTP/REST API data loading

## Installation

Run the installation script:
\`\`\`bash
./install.sh
\`\`\`

Or manually copy plugins to your desired directory and set:
\`\`\`bash
export FLEXCORE_PLUGIN_DIR=/path/to/plugins
\`\`\`

## Usage

Configure in your FlexCore config:
\`\`\`json
{
    "plugin_directory": "/path/to/plugins",
    "enabled_plugins": ["postgres-extractor", "json-transformer", "api-loader"]
}
\`\`\`

## Build Information

- Build time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
- Go version: $(go version)
- Plugins built: $success_count/$total_count

EOF

# Summary
echo -e "\n📊 Build Summary:"
echo -e "   ✅ Successfully built: ${GREEN}$success_count${NC}/$total_count plugins"
echo -e "   📂 Distribution directory: $DIST_DIR"

if [ $success_count -eq $total_count ]; then
    echo -e "\n${GREEN}🎉 All plugins built successfully!${NC}"
    echo -e "\n📦 To install plugins, run:"
    echo -e "   cd $DIST_DIR && ./install.sh"
    exit 0
else
    echo -e "\n${YELLOW}⚠️  Some plugins failed to build${NC}"
    exit 1
fi