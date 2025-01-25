#!/bin/bash

# First, publish the image
echo "Publishing image to Docker Hub..."
./publish.sh
if [ $? -ne 0 ]; then
    echo "Failed to publish image"
    exit 1
fi

# Set variables
KUBE_CONTEXT="gke_morpheus-dev_us-central1_morpheus-cluster"

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
