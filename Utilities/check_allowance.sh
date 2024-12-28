#!/bin/bash

# Check input parameters
if [ $# -lt 2 ] || [ $# -gt 3 ]; then
    echo "Usage: $0 <provider_url> <spender_address> [--json] [--quiet]"
    echo "Example: $0 http://localhost:8082 0xdb008B7F7d8e17F86C747fBFd250af803e42c77c"
    exit 1
fi

PROVIDER_URL="$1"
SPENDER_ADDRESS="$2"
JSON_OUTPUT=0
[ "$3" = "--json" ] && JSON_OUTPUT=1

# Add verbose output flag
VERBOSE=1
[ "$3" = "--quiet" ] && VERBOSE=0

# Check if WALLET_PRIVATE_KEY is set
if [ -z "$WALLET_PRIVATE_KEY" ]; then
    echo "Error: WALLET_PRIVATE_KEY environment variable is not set"
    exit 1
fi

# Extract the API base URL from PROVIDER_URL
API_BASE_URL=$(echo "$PROVIDER_URL" | sed -E 's#(https?://[^/]+).*#\1#')

if [ $VERBOSE -eq 1 ]; then
    echo "=== Allowance Check Details ==="
    echo "Provider URL: $PROVIDER_URL"
    echo "API Base URL: $API_BASE_URL"
    echo "Spender Address: $SPENDER_ADDRESS"
    echo "API Key Length: ${#WALLET_PRIVATE_KEY} characters"
    echo "============================"
    echo "Making API request..."
fi

# Make the allowance request
RESPONSE=$(curl -v -X GET "${API_BASE_URL}/blockchain/allowance?spender=${SPENDER_ADDRESS}" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${WALLET_PRIVATE_KEY}" 2>&1)

# Check if the request was successful and contains allowance
if echo "$RESPONSE" | grep -q "\"allowance\":"; then
    ALLOWANCE=$(echo "$RESPONSE" | grep -o '"allowance":"[^"]*"' | cut -d'"' -f4)
    DECIMAL_ALLOWANCE=$(echo "scale=18; $ALLOWANCE / 1000000000000000000" | bc)
    
    if [ $VERBOSE -eq 1 ]; then
        echo "=== Response Details ==="
        echo "HTTP Status: $(echo "$RESPONSE" | grep "< HTTP" | cut -d' ' -f3)"
        echo "Content-Type: $(echo "$RESPONSE" | grep "< Content-Type:" | cut -d' ' -f3)"
        echo "Raw Response: $RESPONSE"
        echo "===================="
        echo "=== Parsed Values ==="
        echo "Raw Allowance: $ALLOWANCE wei"
        echo "Decimal Allowance: $DECIMAL_ALLOWANCE MOR"
        echo "===================="
    else
        echo "$DECIMAL_ALLOWANCE"
    fi
    exit 0
else
    if [ $VERBOSE -eq 1 ]; then
        echo "=== Error Details ==="
        echo "Failed to get allowance"
        echo "Raw Response: $RESPONSE"
        echo "HTTP Status: $(echo "$RESPONSE" | grep "< HTTP" | cut -d' ' -f3)"
        echo "===================="
    else
        echo "Error: $RESPONSE"
    fi
    exit 1
fi
