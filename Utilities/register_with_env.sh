#!/bin/bash

# Get the directory of the current script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ENV_FILE="$SCRIPT_DIR/.env"  # Changed path to look in current directory
REGISTER_SCRIPT="$SCRIPT_DIR/register_provider.sh"

# Add debug output
echo "Debug: Looking for .env file at: $ENV_FILE"

# Check if register_provider.sh exists and is executable
if [ ! -f "$REGISTER_SCRIPT" ]; then
    echo "Error: register_provider.sh not found"
    exit 1
fi

if [ ! -x "$REGISTER_SCRIPT" ]; then
    echo "Error: register_provider.sh is not executable"
    echo "Please run: chmod +x $REGISTER_SCRIPT"
    exit 1
fi

# Source the .env file
if [ -f "$ENV_FILE" ]; then
    echo "Debug: Found .env file, sourcing it now"
    set -a
    source "$ENV_FILE"
    set +a
    
    # Add debug output after sourcing
    echo "Debug: After sourcing .env:"
    echo "MODEL_ID=$MODEL_ID"
    echo "WALLET_ADDRESS=$WALLET_ADDRESS"
    echo "WALLET_PRIVATE_KEY=$WALLET_PRIVATE_KEY"
    echo "PROVIDER_URL=$PROVIDER_URL"
    echo "PROVIDER_STAKE=$PROVIDER_STAKE"
else
    echo "Error: .env file not found at $ENV_FILE"
    exit 1
fi

# Check if required variables (excluding PROVIDER_STAKE) are set
for var in "MODEL_ID" "WALLET_ADDRESS" "WALLET_PRIVATE_KEY" "PROVIDER_URL"; do
    if [ -z "${!var}" ]; then
        echo "Error: $var is not set in .env file"
        exit 1
    fi
done

# Validate PROVIDER_URL format and extract endpoint
if [[ ! "$PROVIDER_URL" =~ ^https?:// ]]; then
    echo "Error: PROVIDER_URL must start with http:// or https://"
    echo "Current value: $PROVIDER_URL"
    exit 1
fi

# Verify PROVIDER_URL contains hostname:port
if [[ ! "$PROVIDER_URL" =~ :[0-9]+(/|$) ]]; then
    echo "Error: PROVIDER_URL must include port number"
    echo "Current value: $PROVIDER_URL"
    exit 1
fi

# Optional stake defaults to 0.2 MOR if undefined
MIN_STAKE="200000000000000000"
if [ -z "$PROVIDER_STAKE" ]; then
    echo "Debug: PROVIDER_STAKE not set, defaulting to $MIN_STAKE"
    PROVIDER_STAKE="$MIN_STAKE"
elif [ "$PROVIDER_STAKE" -lt "$MIN_STAKE" ]; then
    echo "Debug: PROVIDER_STAKE below minimum, overriding with $MIN_STAKE"
    PROVIDER_STAKE="$MIN_STAKE"
fi

# Call the register_provider script with the environment variables
"$REGISTER_SCRIPT" "$MODEL_ID" "$WALLET_ADDRESS" "$WALLET_PRIVATE_KEY" "$PROVIDER_URL" "$PROVIDER_STAKE"
