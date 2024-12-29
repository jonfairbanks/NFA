#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

# Check for required environment variables
if [ -z "$PROVIDER_WALLET_PRIVATE_KEY" ] || [ -z "$CONSUMER_WALLET_PRIVATE_KEY" ]; then
    echo "Error: Missing required wallet private keys in .env file"
    exit 1
fi

echo "Pushing images..."
./push-images.sh
if [ $? -ne 0 ]; then
    echo "Image push failed"
    exit 1
fi

echo "Deploying provider components..."
./deploy-provider.sh
if [ $? -ne 0 ]; then
    echo "Provider deployment failed"
    exit 1
fi

echo "Deploying consumer components..."
./deploy-consumer.sh
if [ $? -ne 0 ]; then
    echo "Consumer deployment failed"
    exit 1
fi

echo "All components deployed successfully!"
