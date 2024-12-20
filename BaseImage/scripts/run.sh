#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${SCRIPT_DIR}/utils/env_loader.sh"
source "${SCRIPT_DIR}/utils/docker_helper.sh"

# Load environment variables
load_env || {
    echo "Error: No .env file found"
    exit 1
}

# Default values
DEFAULT_MARKETPLACE_HOST="localhost"
DEFAULT_MARKETPLACE_PORT="9000"
DEFAULT_PORT=8080

# Check if port is in use
check_port() {
    local port=$1
    if lsof -i :"$port" > /dev/null 2>&1; then
        echo "Port $port is already in use. Please choose a different port."
        return 1
    fi
    return 0
}

# Parse marketplace URL or use defaults
MARKETPLACE_HOST=${MARKETPLACE_HOST:-$DEFAULT_MARKETPLACE_HOST}
MARKETPLACE_PORT=${MARKETPLACE_PORT:-$DEFAULT_MARKETPLACE_PORT}
MARKETPLACE_URL="http://${MARKETPLACE_HOST}:${MARKETPLACE_PORT}"

# Use custom port if specified
PORT=${PORT:-$DEFAULT_PORT}

# Check if marketplace is accessible
echo "Checking marketplace connectivity..."
if ! curl -s "${MARKETPLACE_URL}/health" > /dev/null; then
    echo "Warning: Marketplace not accessible at ${MARKETPLACE_URL}"
    echo "Please ensure Morpheus Node is running at the correct port"
    read -p "Continue anyway? (y/N): " continue_anyway
    if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if default port is available
if ! check_port "$PORT"; then
    # Try to find an available port
    for try_port in {8081..8100}; do
        if check_port "$try_port"; then
            echo "Found available port: $try_port"
            PORT=$try_port
            break
        fi
    done
    if [ "$PORT" = "$DEFAULT_PORT" ]; then
        echo "No available ports found in range 8080-8100"
        exit 1
    fi
fi

echo "Starting container with:"
echo "MARKETPLACE_URL: $MARKETPLACE_URL"
echo "PORT: $PORT"

# Stop any existing container
docker rm -f nfa-proxy 2>/dev/null || true

# Run with explicit port binding instead of host networking
docker run -d \
  -p "${PORT}:${PORT}" \
  -e "MARKETPLACE_URL=${MARKETPLACE_URL}/v1/chat/completions" \
  -e "PORT=${PORT}" \
  -e "SESSION_DURATION=1h" \
  --name nfa-proxy \
  nfa-base

# Add container health check
echo "Waiting for container to start..."
for i in {1..30}; do
    if curl -s "http://localhost:${PORT}/health" > /dev/null; then
        echo "Container is running and healthy"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Container failed to start properly. Checking logs:"
        docker logs nfa-proxy
        exit 1
    fi
    sleep 1
done

echo "Container started. To view logs:"
echo "docker logs nfa-proxy -f"