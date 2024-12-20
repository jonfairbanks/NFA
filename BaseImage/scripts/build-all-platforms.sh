#!/bin/bash

set -e  # Exit on error

# Configuration
VERSION=${VERSION:-"latest"}
REGISTRY=${REGISTRY:-"srt0422"}
PLATFORMS="linux/amd64,linux/arm64"
NFA_PROXY_IMAGE="openai-morpheus-proxy"
MARKETPLACE_IMAGE="morpheus-marketplace"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

build_and_push() {
    local context=$1
    local dockerfile=$2
    local image=$3
    local platforms=$4
    local tags=$5

    log "Building ${image} for platforms: ${platforms}"
    echo "Tags: ${tags}"

    # Create tag arguments
    local tag_args=""
    for tag in ${tags}; do
        tag_args="${tag_args} -t ${REGISTRY}/${image}:${tag}"
    done

    # Build multi-platform image
    docker buildx build \
        --platform ${platforms} \
        --push \
        ${tag_args} \
        -f ${dockerfile} \
        ${context}

    log "${GREEN}Successfully built and pushed ${image}${NC}"
}

# Ensure buildx is available and using modern builder
setup_buildx() {
    log "Setting up docker buildx"
    docker buildx create --use --name multiarch-builder --platform ${PLATFORMS} || true
    docker buildx inspect --bootstrap
}

# Main build process
main() {
    log "Starting multi-platform build process"
    setup_buildx

    # Build NFA Proxy
    build_and_push \
        "." \
        "Dockerfile.proxy" \
        "${NFA_PROXY_IMAGE}" \
        "${PLATFORMS}" \
        "latest beta-1"

    # Build Marketplace (proxy-router)
    build_and_push \
        "./morpheus-node/proxy-router" \
        "Dockerfile" \
        "${MARKETPLACE_IMAGE}" \
        "${PLATFORMS}" \
        "latest beta-1"

    log "${GREEN}All builds completed successfully${NC}"
}

# Run main function
main

# Print success message with usage instructions
echo -e """
${GREEN}Build completed successfully!${NC}

To use these images in your docker-compose.yml:

  marketplace:
    image: ${REGISTRY}/${MARKETPLACE_IMAGE}:beta-1
    platform: linux/amd64  # or linux/arm64

  nfa-proxy:
    image: ${REGISTRY}/${NFA_PROXY_IMAGE}:beta-1
    platform: linux/amd64  # or linux/arm64

To pull the images:
  docker pull ${REGISTRY}/${NFA_PROXY_IMAGE}:beta-1
  docker pull ${REGISTRY}/${MARKETPLACE_IMAGE}:beta-1
"""
