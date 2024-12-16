#!/bin/zsh

PORT=8080
MODEL_ID=${MODEL_ID:-$(grep MODEL_ID ../.env | cut -d '=' -f2)}

# Verify MODEL_ID is available
if [ -z "$MODEL_ID" ]; then
    echo "Error: MODEL_ID not set"
    exit 1
fi

echo "Using MODEL_ID: $MODEL_ID"

# Function to check if a service is running on a port
check_service() {
    local port=$1
    if ! nc -z localhost $port 2>/dev/null; then
        return 1
    fi
    return 0
}

# Check if proxy is running
if ! check_service $PORT; then
    echo "Error: Proxy is not running on port $PORT"
    echo "Please start it with docker compose up"
    docker ps
    echo "Container logs:"
    docker logs nfa-proxy
    exit 1
fi

# Check if marketplace is running
if ! check_service 9000; then
    echo "Error: Marketplace is not running on port 9000"
    echo "Please start it with docker compose up"
    exit 1
fi

echo "Testing non-streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'"$MODEL_ID"'",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

echo -e "\nTesting streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "0x6a4813e866a48da528c533e706344ea853a1d3f21e37b4c8e7ffd5ff25772024",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'