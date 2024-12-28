#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <model_id> <price_per_second> <provider_url>"
  echo "Example: $0 0xe1e6e3e77148d140065ef2cd4fba7f4ae59c90e1639184b6df5c84 10000000000 http://localhost:8082"
  exit 1
fi

MODEL_ID=$1
PRICE_PER_SECOND=$2
PROVIDER_URL=$3

# Validate minimum price per second
MIN_PRICE=10000000000  # 0.00000001 MOR
if [ "$PRICE_PER_SECOND" -lt "$MIN_PRICE" ]; then
    echo "Error: Price per second must be at least $MIN_PRICE (0.00000001 MOR)"
    exit 1
fi

# Create the bid using the blockchain API
response=$(curl -s -w "\n%{http_code}" -X POST "$PROVIDER_URL/blockchain/bids" \
  -H "Content-Type: application/json" \
  -d '{
    "modelId": "'"$MODEL_ID"'",
    "pricePerSecond": "'"$PRICE_PER_SECOND"'"
  }')

# Get status code from response
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    echo "Bid creation successful!"
    echo "$body"
    exit 0
else
    echo "Bid creation failed with status code: $http_code"
    echo "Response: $body"
    exit 1
fi
