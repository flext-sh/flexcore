#!/bin/bash

# FlexCore Deployment Master - Unified deployment system
# Eliminates duplication among multiple deployment scripts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Deployment configuration
DEPLOYMENT_TYPE="${1:-local}"
ENVIRONMENT="${ENVIRONMENT:-development}"
REPLICAS="${REPLICAS:-1}"

echo -e "${BLUE}🚀 FLEXCORE UNIFIED DEPLOYMENT SYSTEM${NC}"
echo "======================================"
echo "Type: $DEPLOYMENT_TYPE"
echo "Environment: $ENVIRONMENT"
echo "Replicas: $REPLICAS"
echo "Project: $PROJECT_ROOT"
echo ""

# Deployment counters
TOTAL_DEPLOYMENTS=0
SUCCESSFUL_DEPLOYMENTS=0
FAILED_DEPLOYMENTS=0
declare -a FAILED_DEPLOYMENT_NAMES=()

# Utility functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((SUCCESSFUL_DEPLOYMENTS++))
    ((TOTAL_DEPLOYMENTS++))
}

log_failure() {
    echo -e "${RED}[FAILURE]${NC} $1"
    FAILED_DEPLOYMENT_NAMES+=("$1")
    ((FAILED_DEPLOYMENTS++))
    ((TOTAL_DEPLOYMENTS++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Deployment execution wrapper
run_deployment() {
    local deploy_name="$1"
    local deploy_command="$2"
    local timeout="${3:-300}"
    
    log_info "Deploying: $deploy_name"
    
    if timeout "${timeout}s" bash -c "$deploy_command"; then
        log_success "$deploy_name"
        return 0
    else
        log_failure "$deploy_name"
        return 1
    fi
}

# Environment validation
validate_deployment_environment() {
    echo "🔍 Deployment Environment Validation"
    echo "-----------------------------------"
    
    cd "$PROJECT_ROOT"
    
    # Check required files
    local required_files=("Makefile" "docker-compose.yml")
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ] && [ ! -f "deployments/docker/development/$file" ]; then
            log_failure "Missing required file: $file"
            return 1
        fi
    done
    
    # Check Docker availability for container deployments
    if [[ "$DEPLOYMENT_TYPE" == *"docker"* ]] || [[ "$DEPLOYMENT_TYPE" == *"cluster"* ]]; then
        if ! command -v docker >/dev/null 2>&1; then
            log_failure "Docker not available for container deployment"
            return 1
        fi
        
        if ! command -v docker-compose >/dev/null 2>&1; then
            log_failure "Docker Compose not available"
            return 1
        fi
    fi
    
    # Check native build dependencies for native deployments
    if [[ "$DEPLOYMENT_TYPE" == *"native"* ]] || [[ "$DEPLOYMENT_TYPE" == "local" ]]; then
        if [ ! -f "third_party/windmill/windmill-backend" ]; then
            log_failure "Windmill backend binary not found - run 'make windmill' first"
            return 1
        fi
    fi
    
    log_success "Deployment environment validated"
    echo ""
}

# Local development deployment
deploy_local() {
    echo "💻 Local Development Deployment"
    echo "------------------------------"
    
    cd "$PROJECT_ROOT"
    
    # Build if needed
    run_deployment "Local build" "make build-dev" 180
    
    # Start local development
    case "$ENVIRONMENT" in
        "windmill")
            run_deployment "Local Windmill dev" "make dev-windmill" 60
            ;;
        "full")
            run_deployment "Local full stack" "make dev-full" 60
            ;;
        *)
            run_deployment "Local development" "make dev" 60
            ;;
    esac
    
    echo ""
}

# Docker deployment
deploy_docker() {
    echo "🐳 Docker Deployment"
    echo "------------------"
    
    cd "$PROJECT_ROOT"
    
    # Build Docker images
    case "$ENVIRONMENT" in
        "production")
            run_deployment "Docker production build" "make docker-prod" 600
            ;;
        *)
            run_deployment "Docker development build" "make docker-dev" 400
            ;;
    esac
    
    # Start containers
    local compose_file="deployments/docker/${ENVIRONMENT}/docker-compose.yml"
    if [ ! -f "$compose_file" ]; then
        compose_file="deployments/docker/development/docker-compose.yml"
    fi
    
    run_deployment "Docker compose up" "docker-compose -f $compose_file up -d" 120
    
    echo ""
}

# Distributed cluster deployment
deploy_cluster() {
    echo "🌐 Distributed Cluster Deployment"
    echo "--------------------------------"
    
    cd "$PROJECT_ROOT"
    
    # Use existing cluster scripts
    if [ -f "scripts/deployment/start-real-cluster.sh" ]; then
        run_deployment "Distributed cluster" "./scripts/deployment/start-real-cluster.sh" 300
    else
        run_deployment "Docker cluster" "docker-compose -f deployments/docker/distributed/docker-compose.yml up -d --scale worker=$REPLICAS" 180
    fi
    
    echo ""
}

# Native deployment (without Docker)
deploy_native() {
    echo "⚡ Native Deployment"
    echo "------------------"
    
    cd "$PROJECT_ROOT"
    
    # Ensure native builds are complete
    run_deployment "Windmill backend validation" "make windmill-test" 60
    run_deployment "Native system validation" "make windmill-validate-native" 120
    
    # Start native processes
    log_info "Starting native Windmill backend..."
    ./third_party/windmill/windmill-backend &
    WINDMILL_PID=$!
    
    # Start FlexCore application
    if [ -f "build/flexcore" ]; then
        log_info "Starting FlexCore application..."
        ./build/flexcore &
        FLEXCORE_PID=$!
    fi
    
    # Store PIDs for cleanup
    echo "$WINDMILL_PID" > /tmp/windmill.pid
    [ -n "$FLEXCORE_PID" ] && echo "$FLEXCORE_PID" > /tmp/flexcore.pid
    
    log_success "Native deployment started"
    log_info "Windmill PID: $WINDMILL_PID"
    [ -n "$FLEXCORE_PID" ] && log_info "FlexCore PID: $FLEXCORE_PID"
    
    echo ""
}

# Health check after deployment
health_check() {
    echo "🩺 Deployment Health Check"
    echo "-------------------------"
    
    cd "$PROJECT_ROOT"
    
    # Wait for services to start
    sleep 10
    
    # Check Windmill
    if curl -s http://localhost:8000/api/version >/dev/null 2>&1; then
        log_success "Windmill API accessible"
    else
        log_warning "Windmill API not accessible (may still be starting)"
    fi
    
    # Check FlexCore
    if [ -f "build/flexcore" ] && pgrep -f flexcore >/dev/null; then
        log_success "FlexCore application running"
    else
        log_warning "FlexCore application not detected"
    fi
    
    # Check Docker containers if applicable
    if [[ "$DEPLOYMENT_TYPE" == *"docker"* ]] || [[ "$DEPLOYMENT_TYPE" == *"cluster"* ]]; then
        local running_containers=$(docker ps --format "table {{.Names}}" | grep -v NAMES | wc -l)
        if [ "$running_containers" -gt 0 ]; then
            log_success "Docker containers running: $running_containers"
        else
            log_failure "No Docker containers running"
        fi
    fi
    
    echo ""
}

# Stop deployment
stop_deployment() {
    echo "🛑 Stopping Deployment"
    echo "---------------------"
    
    cd "$PROJECT_ROOT"
    
    # Stop Docker containers
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose down 2>/dev/null || true
    fi
    
    # Stop native processes
    if [ -f "/tmp/windmill.pid" ]; then
        local windmill_pid=$(cat /tmp/windmill.pid)
        kill "$windmill_pid" 2>/dev/null || true
        rm -f /tmp/windmill.pid
    fi
    
    if [ -f "/tmp/flexcore.pid" ]; then
        local flexcore_pid=$(cat /tmp/flexcore.pid)
        kill "$flexcore_pid" 2>/dev/null || true
        rm -f /tmp/flexcore.pid
    fi
    
    # Use existing stop script if available
    if [ -f "scripts/deployment/stop-cluster.sh" ]; then
        ./scripts/deployment/stop-cluster.sh
    fi
    
    log_success "Deployment stopped"
    echo ""
}

# Main deployment execution
main() {
    log_info "Starting FlexCore unified deployment system"
    echo ""
    
    # Handle stop command
    if [ "$1" = "stop" ]; then
        stop_deployment
        exit 0
    fi
    
    # Environment validation
    if ! validate_deployment_environment; then
        log_failure "Deployment environment validation failed"
        exit 1
    fi
    
    # Execute deployment based on type
    case "$DEPLOYMENT_TYPE" in
        "local")
            deploy_local
            ;;
        "docker")
            deploy_docker
            ;;
        "cluster"|"distributed")
            deploy_cluster
            ;;
        "native")
            deploy_native
            ;;
        *)
            log_failure "Unknown deployment type: $DEPLOYMENT_TYPE"
            exit 1
            ;;
    esac
    
    # Health check
    health_check
    
    # Results summary
    echo "📊 DEPLOYMENT RESULTS SUMMARY"
    echo "============================"
    echo "Total Deployments: $TOTAL_DEPLOYMENTS"
    echo "Successful: $SUCCESSFUL_DEPLOYMENTS"
    echo "Failed: $FAILED_DEPLOYMENTS"
    echo ""
    
    if [ ${#FAILED_DEPLOYMENT_NAMES[@]} -gt 0 ]; then
        echo -e "${RED}❌ FAILED DEPLOYMENTS:${NC}"
        for deployment in "${FAILED_DEPLOYMENT_NAMES[@]}"; do
            echo "  • $deployment"
        done
        echo ""
        exit 1
    else
        echo -e "${GREEN}🎉 DEPLOYMENT SUCCESSFUL!${NC}"
        echo -e "${GREEN}✅ FlexCore deployed successfully${NC}"
        echo -e "${GREEN}🌐 Services are running and accessible${NC}"
        
        # Show access information
        echo ""
        echo "📡 ACCESS INFORMATION"
        echo "===================="
        echo "Windmill UI: http://localhost:8000"
        echo "FlexCore API: http://localhost:3001 (if running)"
        echo ""
        echo "To stop deployment: $0 stop"
        
        exit 0
    fi
}

# Usage information
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "FlexCore Unified Deployment System"
    echo ""
    echo "Usage: $0 [type] [stop]"
    echo ""
    echo "Deployment Types:"
    echo "  local (default)  - Local development environment"
    echo "  docker          - Docker containerized deployment"
    echo "  cluster         - Distributed cluster deployment"
    echo "  native          - Native processes without Docker"
    echo "  stop            - Stop all deployments"
    echo ""
    echo "Environment Variables:"
    echo "  ENVIRONMENT     - development/production/windmill/full"
    echo "  REPLICAS        - Number of worker replicas (default: 1)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Local development"
    echo "  $0 docker                    # Docker deployment"
    echo "  ENVIRONMENT=production $0 docker  # Production Docker"
    echo "  REPLICAS=3 $0 cluster        # 3-node cluster"
    echo "  $0 native                    # Native processes"
    echo "  $0 stop                      # Stop all"
    exit 0
fi

# Execute main function
main "$@"