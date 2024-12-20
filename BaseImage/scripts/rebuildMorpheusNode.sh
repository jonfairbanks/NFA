#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${SCRIPT_DIR}/utils/env_loader.sh"

# Ensure base .env exists
create_base_env "${SCRIPT_DIR}/.."

# Load environment variables from BaseImage
load_env || {
    echo "Error: Failed to load environment variables"
    exit 1
}

# Verify required environment variables
validate_env "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" || exit 1
PORT=$MARKETPLACE_PORT 
echo "The PORT variable is set to ${PORT}"
# Default values
DEFAULT_PORT=9000
PORT=${PORT:-$DEFAULT_PORT}
DEFAULT_CLONE_DIR="morpheus-node"
CLONE_DIR=${1:-$DEFAULT_CLONE_DIR}

# Check if directory exists
if [ ! -d "$CLONE_DIR" ]; then
    echo "Error: Directory $CLONE_DIR not found. Please run runMorpheusNode.sh first."
    exit 1
fi

# Detect host architecture and set platform
HOST_ARCH=$(uname -m)
case "${HOST_ARCH}" in
    x86_64)  PLATFORM="linux/amd64" ;;
    aarch64|arm64) PLATFORM="linux/arm64" ;;
    *)       
        echo "Unsupported architecture: ${HOST_ARCH}"
        exit 1
        ;;
esac

# Navigate to the project directory
cd "$CLONE_DIR/proxy-router" || { echo "Failed to navigate to proxy-router directory"; exit 1; }

# Check proxy-router env before any operations
ensure_env_exists ".env" || exit 1

# Setup config files from examples
echo "Setting up configuration files..."

# Setup .env file from example
if [ ! -f ".env" ]; then
    if [ ! -f ".env.example" ]; then
        echo "Error: .env.example not found in proxy-router directory"
        exit 1
    fi
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

# Setup models-config.json from example
if [ ! -f "models-config.json" ]; then
    if [ ! -f "models-config.json.example" ]; then
        echo "Error: models-config.json.example not found"
        exit 1
    fi
    echo "Creating models-config.json from example..."
    cp models-config.json.example models-config.json
fi

# Update the required variables using the modified set_env_var function
set_env_var ".env" "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WEB_ADDRESS" "0.0.0.0:${PORT}"
set_env_var ".env" "WEB_PUBLIC_URL" "http://localhost:${PORT}"

# Verify config files exist and have content
if [ ! -f .env ] || [ ! -s .env ] || [ ! -f models-config.json ] || [ ! -s models-config.json ]; then
    echo "Error: One or more configuration files are missing or empty"
    exit 1
fi

echo "Configuration files are ready."

echo "Environment file contents:"
cat .env

# Stop existing containers
echo "Stopping existing containers..."
docker compose down

# Rebuild the container with platform specification
echo "Rebuilding Docker containers for platform ${PLATFORM}..."
DOCKER_DEFAULT_PLATFORM="${PLATFORM}" \
HTTP_PORT="${PORT}" \
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}" \
docker compose build

# Start the containers
echo "Starting containers..."
DOCKER_DEFAULT_PLATFORM="${PLATFORM}" HTTP_PORT="${PORT}" docker compose up -d

# Add health check
echo "Waiting for service to be healthy..."
for i in {1..30}; do
    if docker compose logs | grep -q "http server is listening"; then
        echo "Service is healthy!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Service failed to become healthy. Logs:"
        docker compose logs
        exit 1
    fi
    echo "Attempt $i of 30..."
    sleep 2
done

# Provide status information
echo "Morpheus Node has been rebuilt and restarted"
echo "HTTP API is exposed on port $PORT"
echo "You can access the API at http://localhost:$PORT"

# Show logs
echo "To view logs, run: docker compose logs -f"