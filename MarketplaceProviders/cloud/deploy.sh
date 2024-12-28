#!/bin/bash
set -e

# Load environment variables
source .env

# Define remaining constants
REGISTRY="srt0422"
PROVIDER_IMAGE="${IMAGE_NAME}"
# VERSION is now from .env

# Push images to registry
echo "Pushing images to registry..."
docker push ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION}
docker push ${REGISTRY}/${PROVIDER_IMAGE}:latest

echo "Using image: ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION}"

# Create PVC definition
cat << EOF > provider-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: provider-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

# Check if cluster exists
if ! gcloud container clusters describe ${CLUSTER_NAME} \
    --region ${REGION} \
    --project ${PROJECT_ID} >/dev/null 2>&1; then
    echo "Creating new cluster ${CLUSTER_NAME}"
    gcloud container clusters create ${CLUSTER_NAME} \
        --region ${REGION} \
        --project ${PROJECT_ID} \
        --num-nodes 1 \
        --machine-type e2-standard-2 || exit 1

    # Create a node pool with adequate resources
    gcloud container node-pools create "provider-pool" \
        --cluster ${CLUSTER_NAME} \
        --region ${REGION} \
        --machine-type e2-standard-2 \
        --num-nodes 1 \
        --node-labels=pool=provider
else
    echo "Cluster ${CLUSTER_NAME} already exists"
fi

# Get credentials for kubectl
gcloud container clusters get-credentials ${CLUSTER_NAME} \
    --region ${REGION} \
    --project ${PROJECT_ID}

# Create Kubernetes secret for wallet private key
WALLET_PRIVATE_KEY="74fa5aa80d2bda8f9c2f8c651eed32a2e69868fbe146ace58a567990bb822a85"
kubectl create secret generic provider-secrets \
    --from-literal=wallet-private-key=${WALLET_PRIVATE_KEY} \
    --dry-run=client -o yaml | kubectl apply -f -

# Check if PVC exists, create only if missing
if ! kubectl get pvc provider-data >/dev/null 2>&1; then
    echo "Creating new PVC provider-data..."
    kubectl apply -f provider-pvc.yaml
else
    echo "Using existing PVC provider-data"
fi

# Update image version in deployment yaml
sed -i '' "s|image: ${REGISTRY}/${PROVIDER_IMAGE}:.*|image: ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION}|" provider-deployment.yaml

# Deploy
kubectl apply -f provider-deployment.yaml
kubectl apply -f provider-service.yaml

# Force new deployment rollout
echo "Forcing new deployment rollout to ensure latest image is used..."
kubectl rollout restart deployment provider-deployment
kubectl rollout status deployment provider-deployment

# Cleanup
rm provider-pvc.yaml

echo "Deployment complete for ${REGISTRY}/${PROVIDER_IMAGE}:${VERSION}"
echo "Waiting for service to be ready..."
kubectl get pods -l app=provider -w

# After deployment, check pod status and provide diagnostics if pending
echo "Checking pod status..."
sleep 10

POD_NAME=$(kubectl get pods -l app=provider -o jsonpath='{.items[0].metadata.name}')
POD_STATUS=$(kubectl get pod $POD_NAME -o jsonpath='{.status.phase}')

if [ "$POD_STATUS" == "Pending" ]; then
    echo "Pod is still pending. Checking events and details..."
    kubectl describe pod $POD_NAME
    echo "Node status:"
    kubectl get nodes
    echo "Node details:"
    kubectl describe nodes
    
    # Clean up old deployments
    echo "Cleaning up old deployments..."
    OLD_PODS=$(kubectl get pods -l app=provider --field-selector status.phase!=Pending -o jsonpath='{.items[*].metadata.name}')
    if [ ! -z "$OLD_PODS" ]; then
        echo "Removing old pods: $OLD_PODS"
        kubectl delete pods $OLD_PODS --force --grace-period=0
    fi
fi

# Continue with existing watch
kubectl get pods -l app=provider -w