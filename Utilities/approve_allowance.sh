#!/bin/bash

# Check input parameters
if [ $# -ne 3 ]; then
    echo "Usage: $0 <wallet_address> <provider_url> <amount>"
    exit 1
fi

WALLET_ADDRESS="$1"
PROVIDER_URL="$2"
AMOUNT="$3"

# Check if WALLET_PRIVATE_KEY is set
if [ -z "$WALLET_PRIVATE_KEY" ]; then
    echo "Error: WALLET_PRIVATE_KEY environment variable is not set"
    exit 1
fi

# Extract the API base URL from PROVIDER_URL
API_BASE_URL=$(echo "$PROVIDER_URL" | sed -E 's#(https?://[^/]+).*#\1#')

# Debug: Show the curl command that will be executed
echo "Making API call:"
echo "curl -s -X POST \"${API_BASE_URL}/blockchain/approve\" \\
    -H \"Content-Type: application/json\" \\
    -H \"X-API-Key: ${WALLET_PRIVATE_KEY}\" \\
    -d '{
        \"spender\": \"${WALLET_ADDRESS}\",
        \"amount\": \"${AMOUNT}\"
    }'"

# Make the approve request to the blockchain/approve endpoint
RESPONSE=$(curl -s -X POST "${API_BASE_URL}/blockchain/approve" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${WALLET_PRIVATE_KEY}" \
    -d "{
        \"spender\": \"${WALLET_ADDRESS}\",
        \"amount\": \"${AMOUNT}\"
    }")

# Check if the request was successful
if echo "$RESPONSE" | grep -q "error"; then
    echo "Error: Approval failed"
    echo "Response: $RESPONSE"
    exit 1
else
    # Verify the allowance was set
    ALLOWANCE_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/blockchain/allowance?spender=${WALLET_ADDRESS}" \
        -H "X-API-Key: ${WALLET_PRIVATE_KEY}")
    
    if echo "$ALLOWANCE_RESPONSE" | grep -q "\"allowance\":"; then
        echo "Approval successful"
        echo "Response: $ALLOWANCE_RESPONSE"
    else
        echo "Error: Failed to verify allowance"
        echo "Response: $ALLOWANCE_RESPONSE"
        exit 1
    fi
fi