#!/bin/bash

# Source the .env file
if [ -f .env ]; then
    set -a
    source .env
    set +a
else
    echo "Error: .env file not found"
    exit 1
fi

# Check if required environment variables are set
if [ -z "$WALLET_ADDRESS" ] || [ -z "$PROVIDER_URL" ]; then
    echo "Error: Required environment variables not set in .env"
    echo "Required: WALLET_ADDRESS, PROVIDER_URL"
    exit 1
fi

# Validate PROVIDER_URL format
if ! echo "$PROVIDER_URL" | grep -qE '^https?://[^[:space:]]+$'; then
    echo "Error: Invalid PROVIDER_URL format"
    exit 1
fi

# Extract hostname:port from PROVIDER_URL for endpoint
ENDPOINT=$(echo "$PROVIDER_URL" | sed -E 's#^https?://##')

# Set minimum required amounts based on contract requirements
MIN_PROVIDER_STAKE="200000000000000000"  # 0.2 MOR
APPROVAL_AMOUNT="600000000000000000"     # 0.6 MOR for all operations

# Add API key header to all requests
export API_KEY="${WALLET_PRIVATE_KEY}"

# Approve ERC20 token allowance
# echo "Approving token allowance of $APPROVAL_AMOUNT MOR..."
# if ! ./approve_allowance.sh "$WALLET_ADDRESS" "$PROVIDER_URL" "$APPROVAL_AMOUNT"; then
#     echo "Error: Failed to approve allowance"
#     echo "Please check your wallet balance and API key"
#     exit 1
# fi

# echo "Token allowance approved successfully"
# echo "Creating provider with endpoint: $ENDPOINT"

# Create provider using the blockchain/providers endpoint
PROVIDER_RESPONSE=$(curl -s -X POST "${PROVIDER_URL}/blockchain/providers" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${API_KEY}" \
    -d "{
        \"endpoint\": \"${ENDPOINT}\",
        \"stake\": \"${MIN_PROVIDER_STAKE}\"
    }")

if echo "$PROVIDER_RESPONSE" | grep -q "error"; then
    echo "Error: Failed to create provider"
    echo "Response: $PROVIDER_RESPONSE"
    exit 1
else
    echo "Provider created successfully"
    echo "Response: $PROVIDER_RESPONSE"
fi
