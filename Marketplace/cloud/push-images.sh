#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

# Push provider images
echo "Pushing provider image..."
docker push ${REGISTRY}/${PROVIDER_IMAGE_NAME}:${VERSION}
docker push "${REGISTRY}/${PROVIDER_IMAGE_NAME}:latest"

# Push consumer images
echo "Pushing consumer image..."
docker push ${REGISTRY}/${CONSUMER_IMAGE_NAME}:${VERSION}
docker push "${REGISTRY}/${CONSUMER_IMAGE_NAME}:latest"

echo "All images pushed successfully!"
