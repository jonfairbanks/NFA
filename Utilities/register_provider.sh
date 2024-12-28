#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 5 ]; then
  echo "Usage: $0 <model_id> <wallet_address> <wallet_private_key> <provider_url> <provider_stake>"
  exit 1
fi

MODEL_ID=$1
WALLET_ADDRESS=$2
WALLET_PRIVATE_KEY=$3
PROVIDER_URL=$4
PROVIDER_STAKE=$5

# Extract hostname:port from PROVIDER_URL for endpoint
ENDPOINT=$(echo "$PROVIDER_URL" | sed -E 's#^https?://##')

# Register the provider using the provided arguments
response=$(curl -s -w "\n%{http_code}" -X POST "$PROVIDER_URL/blockchain/providers" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "'"$ENDPOINT"'",
    "stake": "'"$PROVIDER_STAKE"'"
  }')

# Get status code from response
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    echo "Registration successful!"
    echo "$body"
    exit 0
else
    echo "Registration failed with status code: $http_code"
    echo "Response: $body"
    exit 1
fi
