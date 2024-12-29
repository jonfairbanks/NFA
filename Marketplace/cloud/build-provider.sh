#!/bin/bash
set -e

# Store initial directory and load environment variables
INITIAL_DIR="$(pwd)"
source "${INITIAL_DIR}/.env"

# Define constants
REGISTRY="srt0422"
BUILD_PLATFORM="linux/amd64"
# VERSION is now from .env
PROVIDER_IMAGE="${IMAGE_NAME}"
REPO_DIR="${INITIAL_DIR}/Morpheus-Lumerin-Node"
PROXY_DIR="${REPO_DIR}/proxy-router"
DATA_DIR="${PROXY_DIR}/data"
WALLET_PRIVATE_KEY="74fa5aa80d2bda8f9c2f8c651eed32a2e69868fbe146ace58a567990bb822a85"

# Function to build the provider
build_provider() {
    cd "${PROXY_DIR}"
    
    # Build Go application
    go install github.com/swaggo/swag/cmd/swag@latest
    make swagger
    chmod +x build.sh && ./build.sh

    # Build and load Docker image
    docker buildx create --use --name provider-builder --platform ${BUILD_PLATFORM} || true
    docker buildx build \
      --platform ${BUILD_PLATFORM} \
      -t ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION} \
      -t ${REGISTRY}/${PROVIDER_IMAGE}:latest \
      --build-arg WALLET_PRIVATE_KEY=${WALLET_PRIVATE_KEY} \
      --load \
      .
}

# Setup workspace
mkdir -p "${DATA_DIR}"

# Ask user for build preference
echo "Select an option:"
echo "1) Update repository and rebuild (full setup)"
echo "2) Skip to build step (use existing configuration)"
read -p "Enter choice [1-2]: " choice

case $choice in
    1)
        if [ -d "${REPO_DIR}" ]; then
            read -p "Repository exists. Update it? (y/n) > " update_repo
            if [[ $update_repo =~ ^[Yy]$ ]]; then
                cd "${REPO_DIR}"
                git fetch origin dev
                git reset --hard origin/dev
                git checkout dev
                git pull
            fi
        else
            git clone https://github.com/Lumerin-protocol/Morpheus-Lumerin-Node.git "${REPO_DIR}"
            cd "${REPO_DIR}"
            git checkout dev
        fi

        # Copy config files if they exist
        cd "${PROXY_DIR}"
        [ -f "models-config.json.example" ] && cp "models-config.json.example" "${DATA_DIR}/models-config.json"
        [ -f "rating-config.json.example" ] && cp "rating-config.json.example" "${DATA_DIR}/rating-config.json"
        
        # Copy .env file to proxy-router.env
        echo "Copying .env file..."
        cp "${PROXY_DIR}/.env" "${INITIAL_DIR}/proxy-router.env" || {
            echo "Error: Failed to copy .env file to proxy-router.env"
            exit 1
        }
        ;;
    2)
        echo "Using existing configuration..."
        
        # Copy .env file to proxy-router.env
        echo "Copying .env file..."
        cp "${PROXY_DIR}/.env" "${INITIAL_DIR}/proxy-router.env" || {
            echo "Error: Failed to copy .env file to proxy-router.env"
            exit 1
        }
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

# Build the provider
build_provider

# Return to initial directory
cd "${INITIAL_DIR}"

echo "Build complete: ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION}"