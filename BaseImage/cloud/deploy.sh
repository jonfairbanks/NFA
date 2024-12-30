#!/bin/bash

# Source version from .env
VERSION=$(grep VERSION ../.env | cut -d '=' -f2)

# Set variables
IMAGE_NAME="openai-morpheus-proxy"
REGISTRY="srt0422"  # Updated to use Docker Hub registry
KUBE_CONTEXT="gke_morpheus-dev_us-central1_morpheus-cluster"

# Tag for Docker Hub is done in build.sh now
echo "Pushing to Docker Hub..."
docker push ${REGISTRY}/${IMAGE_NAME}:${VERSION}
docker push ${REGISTRY}/${IMAGE_NAME}:latest

# Switch to the correct kubernetes context
echo "Switching to kubernetes context..."
kubectl config use-context ${KUBE_CONTEXT}

# Update the deployment image
echo "Updating kubernetes deployment..."
kubectl apply -f nfa-proxy-service.yaml
kubectl apply -f nfa-proxy-deployment.yaml

# Wait for rollout
echo "Waiting for rollout to complete..."
kubectl rollout status deployment/nfa-proxy-deployment

echo "Deployment completed successfully!"
