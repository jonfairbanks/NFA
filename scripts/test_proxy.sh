#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MODEL_ID="0x560d9704d2dba7da8dab2db043f9f8fd9354936561961569ddd874641adee13e"
PROXY_PORT="8081"

# Function to check if Docker is running
check_docker() {
    echo -e "${YELLOW}Checking if Docker is running...${NC}"
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}Docker is not running. Please start Docker first.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Docker is running${NC}"
}

# Function to check and start the NFA proxy container
ensure_proxy_running() {
    echo -e "${YELLOW}Checking NFA proxy container...${NC}"
    
    # Check if container exists and is running
    if [ "$(docker ps -q -f name=nfa-proxy)" ]; then
        echo -e "${GREEN}NFA proxy container is already running${NC}"
        return 0
    fi
    
    # Check if container exists but is stopped
    if [ "$(docker ps -aq -f status=exited -f name=nfa-proxy)" ]; then
        echo -e "${YELLOW}Starting existing NFA proxy container...${NC}"
        docker start nfa-proxy
        sleep 5 # Give container time to start
        return 0
    fi
    
    # Container doesn't exist, need to build and run
    echo -e "${YELLOW}Building and starting new NFA proxy container...${NC}"
    
    # Build the image
    docker build -t nfa-proxy:latest -f Dockerfile.proxy .
    
    # Run the container
    docker run -d \
        --name nfa-proxy \
        -p ${PROXY_PORT}:${PROXY_PORT} \
        -e MARKETPLACE_URL="https://consumer-node-yalzemm5uq-uc.a.run.app" \
        nfa-proxy:latest
    
    # Wait for container to be ready
    sleep 10
}

# Function to check container health
check_health() {
    echo -e "${YELLOW}Checking container health...${NC}"
    
    local max_attempts=5
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${PROXY_PORT}/health)
        if [ "$response" = "200" ]; then
            echo -e "${GREEN}Container is healthy${NC}"
            return 0
        fi
        echo -e "${YELLOW}Attempt $attempt of $max_attempts: Container not ready yet...${NC}"
        sleep 2
        ((attempt++))
    done
    
    echo -e "${RED}Container failed to become healthy${NC}"
    return 1
}

# Function to run tests
run_tests() {
    echo -e "\n${YELLOW}Running tests...${NC}"
    
    # Test 1: Health check
    echo -e "\n${YELLOW}Test 1: Health Check${NC}"
    health_response=$(curl -s http://localhost:${PROXY_PORT}/health)
    echo "Health response: $health_response"
    
    # Test 2: Get available models
    echo -e "\n${YELLOW}Test 2: Get Available Models${NC}"
    models_response=$(curl -s -X GET "http://localhost:${PROXY_PORT}/blockchain/models" \
        -H 'accept: application/json')
    echo "Models response: $models_response"
    
    # Test 3: Create session and test chat completion
    echo -e "\n${YELLOW}Test 3: Creating session and testing chat completion${NC}"
    
    session_response=$(curl -s -X POST "http://localhost:${PROXY_PORT}/blockchain/models/${MODEL_ID}/session" \
        -H 'accept: application/json' \
        -H 'Content-Type: application/json' \
        -d '{"sessionDuration": 600}')
    
    session_id=$(echo $session_response | jq -r '.sessionID')
    
    if [ -n "$session_id" ] && [ "$session_id" != "null" ]; then
        echo "Session created: $session_id"
        
        chat_response=$(curl -s -X POST "http://localhost:${PROXY_PORT}/v1/chat/completions" \
            -H 'accept: application/json' \
            -H 'Content-Type: application/json' \
            -H "session_id: $session_id" \
            -d '{
                "messages": [{"content": "Hello, how are you?", "role": "user"}],
                "stream": true,
                "model": "LMR-Hermes-2-Theta-Llama-3-8B"
            }')
        echo "Chat response: $chat_response"
    else
        echo "Failed to create session. Response: $session_response"
    fi
}

# Function to show container logs
show_logs() {
    echo -e "\n${YELLOW}Recent container logs:${NC}"
    docker logs --tail 50 nfa-proxy
}

# Main execution
echo -e "${YELLOW}Starting test suite...${NC}"

# Run the test sequence
check_docker
ensure_proxy_running
check_health

if [ $? -eq 0 ]; then
    run_tests
    show_logs
else
    echo -e "${RED}Container health check failed. Check the logs:${NC}"
    show_logs
    exit 1
fi 