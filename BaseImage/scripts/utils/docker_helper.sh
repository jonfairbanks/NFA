#!/bin/bash

# Enhanced wait_for_service function with better container checks
wait_for_service() {
    local url=$1
    local container_pattern=$2
    local max_attempts=30
    local attempt=1
    
    echo "Waiting for service at $url..."
    while [ $attempt -le $max_attempts ]; do
        # First check if container is running
        if ! docker ps --format '{{.Names}}' | grep -q "$container_pattern"; then
            echo "Container $container_pattern is not running"
            docker compose logs || true
            return 1
        fi

        if curl -s "$url/health" >/dev/null; then
            echo "Service is healthy!"
            return 0
        fi
        echo "Attempt $attempt of $max_attempts..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "Service health check failed after $max_attempts attempts"
    docker compose logs || true
    return 1
}

# Check if port is available
check_port() {
    local port=$1
    if lsof -i :"$port" > /dev/null 2>&1; then
        echo "Port $port is already in use."
        return 1
    fi
    return 0
}

# Get platform for Docker
get_docker_platform() {
    local host_arch=$(uname -m)
    case "${host_arch}" in
        x86_64)  echo "linux/amd64" ;;
        aarch64|arm64) echo "linux/arm64" ;;
        *)       
            echo "Unsupported architecture: ${host_arch}" >&2
            return 1
            ;;
    esac
}

# Check if a container is running by service name
check_container_running() {
    local service_name=$1
    local container_pattern=$2
    echo "Checking if $service_name is running..."
    
    if docker ps --format '{{.Names}}' | grep -q "$container_pattern"; then
        echo "$service_name container is running"
        return 0
    else
        echo "$service_name container is not running"
        docker ps || true  # Show running containers
        return 1
    fi
}