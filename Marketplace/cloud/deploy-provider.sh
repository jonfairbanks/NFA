#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

echo "Creating provider secrets..."
kubectl create secret generic provider-secrets \
    --from-literal=wallet-private-key=$PROVIDER_WALLET_PRIVATE_KEY \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Creating provider PVC..."
kubectl apply -f provider-pvc.yaml

echo "Deploying provider..."
envsubst < provider-deployment.yaml | kubectl apply -f -
