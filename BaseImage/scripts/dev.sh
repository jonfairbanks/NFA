#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${SCRIPT_DIR}/utils/env_loader.sh"
source "${SCRIPT_DIR}/utils/docker_helper.sh"

# Load environment variables
load_env || {
    echo "Error: No .env file found in current or parent directories"
    exit 1
}

# Default values
DEFAULT_MARKETPLACE_PORT=9000
DEFAULT_PROXY_PORT=8080

MARKETPLACE_PORT=${MARKETPLACE_PORT:-$DEFAULT_MARKETPLACE_PORT}
PROXY_PORT=${PORT:-$DEFAULT_PROXY_PORT}

# Function to check if a port is in use
check_port() {
    lsof -i :"$1" >/dev/null 2>&1
}

# Function to wait for service to be available
wait_for_service() {
    local url=$1
    local max_attempts=30
    local attempt=1
    
    echo "Waiting for service at $url..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url/health" >/dev/null; then
            echo "Service is up!"
            return 0
        fi
        echo "Attempt $attempt of $max_attempts..."
        
        # Update container check to use the proper container name/pattern
        if ! check_container_running "Morpheus Node" "proxy-router"; then
            echo "Container is not running. Logs:"
            docker logs proxy-router || true
            return 1
        fi
        
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "Container logs:"
    docker logs nfa-proxy || true
    return 1
}

# Check for required environment variable
if [ -z "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY:-}" ]; then
    echo "Error: MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY is not set"
    echo "Please ensure this environment variable is set before running the script"
    exit 1
fi

# Check for Morpheus Node
if ! check_port $MARKETPLACE_PORT; then
    echo "Morpheus Node is not running on port $MARKETPLACE_PORT"
    echo "Starting Morpheus Node..."
    
    # Look for runMorpheusNode.sh in the scripts directory
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
    
    # Check if dependencies are already set up by looking for the morpheus-node directory
    if [ -d "${SCRIPT_DIR}/../morpheus-node" ]; then
        echo "Dependencies already set up, rebuilding Morpheus Node..."
        if [ -f "${SCRIPT_DIR}/rebuildMorpheusNode.sh" ]; then
            PORT=$MARKETPLACE_PORT "${SCRIPT_DIR}/rebuildMorpheusNode.sh"
        else
            echo "Error: rebuildMorpheusNode.sh not found in ${SCRIPT_DIR}"
            exit 1
        fi
    else
        # Fresh setup needed
        if [ -f "${SCRIPT_DIR}/runMorpheusNode.sh" ]; then
            "${SCRIPT_DIR}/runMorpheusNode.sh" -p $MARKETPLACE_PORT
        else
            echo "Error: runMorpheusNode.sh not found in ${SCRIPT_DIR}"
            exit 1
        fi
    fi
fi

# Wait for Morpheus Node to be ready
if ! wait_for_service "http://localhost:$MARKETPLACE_PORT"; then
    echo "Error: Morpheus Node failed to start"
    exit 1
fi

# Stop any existing NFA proxy container
echo "Stopping any existing NFA proxy container..."
docker rm -f nfa-proxy 2>/dev/null || true

# Start the NFA proxy container
echo "Starting NFA proxy container..."
docker run -d \
    --name nfa-proxy \
    -p "${PROXY_PORT}:${PROXY_PORT}" \
    -e "MARKETPLACE_URL=http://localhost:${MARKETPLACE_PORT}/v1/chat/completions" \
    -e "PORT=${PROXY_PORT}" \
    -e "SESSION_DURATION=1h" \
    nfa-base

# Use enhanced wait_for_service function
if ! wait_for_service "http://localhost:${PORT}" "proxy-router"; then
    echo "Service failed to become healthy"
    exit 1
fi

# Wait for proxy to be ready
if ! wait_for_service "http://localhost:$PROXY_PORT"; then
    echo "Error: NFA proxy failed to start"
    docker logs nfa-proxy
    exit 1
fi

echo "Development environment is ready:"
echo "- Marketplace Node: http://localhost:$MARKETPLACE_PORT"
echo "- NFA Proxy: http://localhost:$PROXY_PORT"
echo ""
echo "To view proxy logs:"
echo "docker logs -f nfa-proxy"