#!/bin/bash
set -euo pipefail

# Default values
IMAGE_NAME="nfa-base"
DOCKERFILE="Dockerfile.proxy"
BUILD_ARGS=()
PLATFORMS=()

# Detect host architecture
HOST_ARCH=$(uname -m)
case "${HOST_ARCH}" in
    x86_64)  DEFAULT_ARCH="amd64" ;;
    aarch64) DEFAULT_ARCH="arm64" ;;
    arm64)   DEFAULT_ARCH="arm64" ;;
    *)       DEFAULT_ARCH="amd64" ;;
esac

# Help function
usage() {
    echo "Usage: $0 [-t tag] [-f dockerfile] [-p platform] [-a build-arg]"
    echo "  -t: Image tag (default: nfa-base)"
    echo "  -f: Dockerfile to use (default: Dockerfile.proxy)"
    echo "  -p: Platform to build for (default: host architecture)"
    echo "  -a: Build argument in key=value format"
    echo "  -h: Show this help message"
    exit 1
}

# Parse command line arguments
while getopts "t:f:p:a:h" opt; do
    case $opt in
        t) IMAGE_NAME="$OPTARG" ;;
        f) DOCKERFILE="$OPTARG" ;;
        p) PLATFORMS+=("$OPTARG") ;;
        a) BUILD_ARGS+=("--build-arg" "$OPTARG") ;;
        h) usage ;;
        \?) usage ;;
    esac
done

# Load environment variables from .env if present
if [ -f ../.env ]; then
    export $(grep -v '^#' ../.env | xargs)
fi

# If no platform specified, use host architecture
if [ ${#PLATFORMS[@]} -eq 0 ]; then
    PLATFORMS+=("linux/${DEFAULT_ARCH}")
fi

# Construct platform argument
PLATFORM_ARG=""
for platform in "${PLATFORMS[@]}"; do
    if [ -z "$PLATFORM_ARG" ]; then
        PLATFORM_ARG="--platform=${platform}"
    else
        PLATFORM_ARG="${PLATFORM_ARG},${platform}"
    fi
done

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running or not accessible"
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "$DOCKERFILE" ]; then
    echo "Error: Dockerfile '$DOCKERFILE' not found"
    exit 1
fi

echo "Building image ${IMAGE_NAME} for platforms: ${PLATFORMS[*]}"
echo "Using Dockerfile: ${DOCKERFILE}"

# Build the image
# Modified to handle empty BUILD_ARGS
if [ ${#BUILD_ARGS[@]:-0} -eq 0 ]; then
    docker buildx build \
        ${PLATFORM_ARG} \
        -t "${IMAGE_NAME}" \
        -f "${DOCKERFILE}" \
        --load \
        . || {
            echo "Error: Build failed"
            exit 1
        }
else
    docker buildx build \
        ${PLATFORM_ARG} \
        "${BUILD_ARGS[@]}" \
        -t "${IMAGE_NAME}" \
        -f "${DOCKERFILE}" \
        --load \
        . || {
            echo "Error: Build failed"
            exit 1
        }
fi

echo "Build successful: ${IMAGE_NAME}"