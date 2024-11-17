#!/bin/bash
set -euo pipefail

# Default port for the proxy
PORT=${PORT:-8080}

# Function to check if service is healthy
check_service() {
    local port=$1
    local max_attempts=5
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s "http://localhost:${port}/health" > /dev/null; then
            return 0
        fi
        echo "Attempt $attempt: Service not ready, waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done
    return 1
}

# Check if proxy is running and healthy
if ! check_service $PORT; then
    echo "Error: Proxy is not running or not healthy on port $PORT"
    echo "Container logs:"
    docker logs nfa-proxy
    exit 1
fi

# Check if marketplace is running
if ! check_service 9000; then
    echo "Error: Marketplace is not running on port 9000"
    echo "Please start it with ./scripts/runMorpheusNode.sh"
    exit 1
fi

echo "Testing non-streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

echo -e "\nTesting streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'