#!/bin/bash

# Source the .env file
if [ -f .env ]; then
    export $(cat .env | sed 's/#.*//g' | xargs)
else
    echo "Error: .env file not found"
    exit 1
fi

# Check if PROVIDER_URL is set
if [ -z "$PROVIDER_URL" ]; then
    echo "Error: PROVIDER_URL not set in .env"
    exit 1
fi

echo "Checking health endpoint at: $PROVIDER_URL/healthcheck"
echo "----------------------------------------"

# Make health check request with verbose output
response=$(curl -v -s -k -w "\n%{http_code}" "$PROVIDER_URL/healthcheck" 2>&1)

# Get status code from response
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

echo -e "\nResponse Details:"
echo "----------------------------------------"
echo "$body"
echo "----------------------------------------"

if [ "$http_code" -eq 200 ]; then
    echo -e "\n✓ Health check passed (status: $http_code)"
    exit 0
else
    echo -e "\n✗ Health check failed (status: $http_code)"
    exit 1
fi
