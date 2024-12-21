#!/bin/bash

set -e  # Exit on error

# Configuration
REGISTRY=${REGISTRY:-"srt0422"}
PLATFORMS="linux/amd64,linux/arm64"
NFA_PROXY_IMAGE="openai-morpheus-proxy"
MARKETPLACE_IMAGE="morpheus-marketplace"
VERSION_FILE=".version"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

get_next_version() {
    if [ ! -f "$VERSION_FILE" ]; then
        echo "v0.0.0" > "$VERSION_FILE"
        echo "v0.0.0"
        return
    fi

    current_version=$(cat "$VERSION_FILE")
    # Replace Bash-specific regex match with POSIX-compliant grep
    if ! echo "$current_version" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
        echo "v0.0.0" > "$VERSION_FILE"
        echo "v0.0.0"
        return
    fi

    # Extract patch number and increment it
    major=$(echo "$current_version" | cut -d'.' -f1)
    minor=$(echo "$current_version" | cut -d'.' -f2)
    patch=$(echo "$current_version" | cut -d'.' -f3)
    # Remove 'v' from major version
    major_number=${major#v}
    new_patch=$((patch + 1))
    new_version="v${major_number}.${minor}.${new_patch}"
    echo "$new_version" > "$VERSION_FILE"
    echo "$new_version"
}

build_and_push() {
    local context=$1
    local dockerfile=$2
    local image=$3
    local platforms=$4
    local version=$5

    log "Building ${image} version ${version} for platforms: ${platforms}"

    # Build multi-platform image
    docker buildx build \
        --platform ${platforms} \
        --push \
        -t ${REGISTRY}/${image}:${version} \
        -t ${REGISTRY}/${image}:latest \
        -f ${dockerfile} \
        ${context}

    log "${GREEN}Successfully built and pushed ${image}:${version}${NC}"
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

    # Get next version number
    VERSION=$(get_next_version)
    log "Building version: ${VERSION}"

    # Build NFA Proxy
    build_and_push \
        "." \
        "Dockerfile.proxy" \
        "${NFA_PROXY_IMAGE}" \
        "${PLATFORMS}" \
        "${VERSION}"

    # Build Marketplace (proxy-router)
    build_and_push \
        "./morpheus-node/proxy-router" \
        "./morpheus-node/proxy-router/Dockerfile" \
        "${MARKETPLACE_IMAGE}" \
        "${PLATFORMS}" \
        "${VERSION}"

    log "${GREEN}All builds completed successfully${NC}"
}

# Run main function
main

# Print success message with usage instructions
cat << EOF
${GREEN}Build completed successfully!${NC}
Version: ${VERSION}

To use these images in your docker-compose.yml:

  marketplace:
    image: ${REGISTRY}/${MARKETPLACE_IMAGE}:${VERSION}
    platform: linux/amd64  # or linux/arm64

  nfa-proxy:
    image: ${REGISTRY}/${NFA_PROXY_IMAGE}:${VERSION}
    platform: linux/amd64  # or linux/arm64

To pull the images:
  docker pull ${REGISTRY}/${NFA_PROXY_IMAGE}:${VERSION}
  docker pull ${REGISTRY}/${MARKETPLACE_IMAGE}:${VERSION}
EOF
