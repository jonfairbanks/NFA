#!/bin/bash

˚
# Check if the correct number of arguments is provided
if [ "$#" -ne 4 ]; then
  echo "Usage: $0 <model_id> <wallet_address> <wallet_private_key> <provider_url>"
  exit 1
fi

MODEL_ID=$1
WALLET_ADDRESS=$2
WALLET_PRIVATE_KEY=$3
PROVIDER_URL=$4

# Register the provider using the provided arguments
curl -X POST "$PROVIDER_URL/register" \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "'"$MODEL_ID"'",
    "wallet_address": "'"$WALLET_ADDRESS"'",
    "wallet_private_key": "'"$WALLET_PRIVATE_KEY"'"
  }'
