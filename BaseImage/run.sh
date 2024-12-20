#!/bin/bash

set -e  # Exit on error

# Function to check if an image exists
check_image() {
    docker image inspect $1 >/dev/null 2>&1
    return $?
}

# Function to log with timestamp
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Check if proxy-router image exists
if ! check_image proxy-router:latest; then
    log "proxy-router image not found, building it first..."
    ./morpheus-node/proxy-router/docker_build.sh --build
    if [ $? -ne 0 ]; then
        log "Error: Failed to build proxy-router image"
        exit 1
    fi
fi

log "Starting docker compose project..."
docker compose up --remove-orphans --build
