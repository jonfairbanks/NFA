#!/bin/bash

# Check if the correct number of arguments is provided (2 or 3)
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "Usage: $0 <provider_id> <new_endpoint> [<new_stake>]"
    echo "Example: $0 0x1234...5678 mycoolnode.domain.com:3333 200000000000000000"
    exit 1
fi

# Validate provider ID
if [[ ! $1 =~ ^0x[0-9a-fA-F]{40}$ ]]; then
    echo "Error: Invalid provider ID format. Must be a hex address starting with 0x"
    exit 1
fi

PROVIDER_ID=$1
NEW_ENDPOINT=$2

# Extract hostname:port from NEW_ENDPOINT
ENDPOINT=$(echo "$NEW_ENDPOINT" | sed -E 's#^https?://##')

# Define default stake if not provided
if [ "$#" -eq 3 ]; then
    # Verify stake is a valid number
    if ! [[ $3 =~ ^[0-9]+$ ]]; then
        echo "Error: Stake must be a valid number"
        exit 1
    fi
    NEW_STAKE=$3
else
    NEW_STAKE=200000000000000000  # Default 0.2 MOR consistent with minimum stake
fi

# Validate the new stake
MIN_STAKE=200000000000000000  # Minimum stake required (0.2 MOR)
if [ "$NEW_STAKE" -lt "$MIN_STAKE" ]; then
    echo "Error: New stake must be at least $MIN_STAKE (0.2 MOR)"
    exit 1
fi

# Debug output
echo "Making POST request to: $PROVIDER_URL/blockchain/providers"
echo "With endpoint: $ENDPOINT"
echo "With stake: $NEW_STAKE"

# Update the provider using the blockchain API
response=$(curl -s -w "\n%{http_code}" -X POST "$PROVIDER_URL/blockchain/providers" \
    -H "Content-Type: application/json" \
    -d '{
    "endpoint": "'"$ENDPOINT"'",
    "stake": "'"$NEW_STAKE"'"
}')

# Get status code from response
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 204 ]; then
    echo "Provider update successful!"
    echo "$body"
    exit 0
else
    echo "Provider update failed with status code: $http_code"
    echo "Response: $body"
    exit 1
fi