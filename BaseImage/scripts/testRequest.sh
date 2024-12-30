#!/bin/zsh

PORT=8080
MODEL_HANDLE="nfa-llama2"  # Changed back to original model handle
# MODEL_HANDLE="Hermes 3 Llama 3.1"
echo "Using MODEL_HANDLE: $MODEL_HANDLE"

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
# if ! check_service 9000; then
#     echo "Error: Marketplace is not running on port 9000"
#     echo "Please start it with docker compose up"
#     exit 1
# fi

# echo "Testing non-streaming request..."
# curl -v -X POST http://localhost:$PORT/v1/chat/completions \
#   -H "Content-Type: application/json" \
#   -d '{
#     "model": "'"$MODEL_HANDLE"'",
#     "messages": [{"role": "user", "content": "Hello"}]
#   }'

echo -e "\nTesting streaming request..."
curl -v -X POST http://34.127.54.11:8080 /v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'