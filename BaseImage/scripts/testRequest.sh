#!/bin/zsh

PORT=8080
MODEL_HANDLE="LMR-Hermes-2-Theta-Llama-3-8B"

echo "Using MODEL_HANDLE: $MODEL_HANDLE"
echo "Checking environment variables..."
echo "MODEL_ID: $MODEL_ID"

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

# Add verbose output for debugging
echo "Testing non-streaming request to http://localhost:$PORT/v1/chat/completions..."
echo "Request body:"
echo '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}]
}'

echo "Testing non-streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

echo -e "\nTesting streaming request..."
curl -v -X POST http://localhost:$PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'