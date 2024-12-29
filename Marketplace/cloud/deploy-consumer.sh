#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

echo "Creating consumer secrets..."
kubectl create secret generic consumer-secrets \
    --from-literal=wallet-private-key=$CONSUMER_WALLET_PRIVATE_KEY \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Creating consumer PVC..."
kubectl apply -f consumer-pvc.yaml

echo "Creating consumer service..."
kubectl apply -f consumer-service.yaml

echo "Deploying consumer..."
envsubst < consumer-deployment.yaml | kubectl apply -f -
