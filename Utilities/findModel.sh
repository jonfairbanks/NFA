#!/bin/zsh

set -e  # Exit on error
set -u  # Error on undefined variables

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    if ! command -v python3 >/dev/null 2>&1; then
        missing_deps+=("python3")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        echo "Error: Missing required dependencies: ${missing_deps[*]}"
        echo "Please install them using:"
        echo "brew install ${missing_deps[*]}"
        exit 1
    fi

    # Check for Python Levenshtein library
    if ! python3 -c "import Levenshtein" 2>/dev/null; then
        echo "Error: Python Levenshtein library not found"
        echo "Please install it using: pip3 install python-Levenshtein"
        exit 1
    fi
}

# Check if a model name was provided
if [ -z "$1" ]; then
    echo "Usage: $0 <model-name>"
    echo "Example: $0 llama"
    exit 1
fi

check_dependencies

MODEL_SEARCH="$1"

# Try different ports in order
PORTS=(9000 8082 8080 3000)
MARKETPLACE_HOST="localhost"
MARKETPLACE_URL=""

echo "Starting marketplace service check..."

# First try docker container name
if curl -s -f "http://marketplace:9000/health" >/dev/null 2>&1; then
    MARKETPLACE_URL="http://marketplace:9000"
    echo "Found marketplace at container hostname"
else
    # Try localhost ports
    for port in "${PORTS[@]}"; do
        echo -n "Checking localhost:${port}... "
        if curl -s -f "http://localhost:${port}/health" >/dev/null 2>&1; then
            MARKETPLACE_URL="http://localhost:${port}"
            echo "success!"
            break
        else
            echo "failed"
        fi
    done
fi

if [ -z "$MARKETPLACE_URL" ]; then
    echo "ERROR: No running marketplace found"
    echo "Checking running services:"
    ps aux | grep -i "morpheus\|proxy" | grep -v grep
    echo ""
    echo "Checking ports:"
    for port in "${PORTS[@]}"; do
        echo "Port ${port}:"
        lsof -i :${port} || echo "  - Not in use"
    done
    exit 1
fi

echo "Found marketplace at: $MARKETPLACE_URL"
echo "Testing models endpoint..."

MODELS_RESPONSE=$(curl -s -f "${MARKETPLACE_URL}/blockchain/models")
if [ $? -ne 0 ]; then
    echo "Error accessing models endpoint"
    curl -v "${MARKETPLACE_URL}/blockchain/models"
    exit 1
fi

echo "Raw response:"
echo "$MODELS_RESPONSE" | jq .

# Stop here for debugging
exit 0

# Create temporary file for storing results
TEMP_RESULTS=$(mktemp)
trap 'rm -f $TEMP_RESULTS' EXIT

# Function to calculate similarity score
calculate_similarity() {
    local s1="$1"
    local s2="$2"
    python3 -c "
import Levenshtein
import sys
score = Levenshtein.ratio('${s1}'.lower(), '${s2}'.lower())
print(f'{score:.4f}')
"
}

echo "Fetching models from marketplace at ${MARKETPLACE_URL}..."

# Enhanced connection testing function
test_marketplace_connection() {
    local url="$1"
    echo "Testing connection to $url..."
    
    # Try with both HTTP and HTTPS
    for protocol in "http" "https"; do
        test_url="${protocol}://${url#*://}"
        echo "Trying $test_url..."
        
        # Use -k to allow insecure connections, -f to fail silently
        local response=$(curl -k -v -m 5 "$test_url/health" 2>&1)
        local curl_exit=$?
        
        echo "Curl exit code: $curl_exit"
        if [ "$DEBUG" = "1" ]; then
            echo "Full response:"
            echo "$response"
        fi

        if [ $curl_exit -eq 0 ]; then
            MARKETPLACE_URL="$test_url"
            echo "✓ Successfully connected to $test_url"
            return 0
        else
            echo "✗ Failed to connect to $test_url (error $curl_exit)"
            echo "Error details:"
            echo "$response" | grep -E "^[*<>]" || true
        fi
    done
    return 1
}

# Try multiple marketplace endpoints
echo "Testing marketplace endpoints..."
FOUND_ENDPOINT=0

# Store original marketplace URL
ORIGINAL_URL="$MARKETPLACE_URL"

# Try original URL first
if test_marketplace_connection "$ORIGINAL_URL"; then
    FOUND_ENDPOINT=1
else
    echo "Failed to connect to configured URL $ORIGINAL_URL"
    echo "Trying alternative ports..."
    
    # Try alternative ports on localhost
    for port in 9000 8082 8080 3000; do
        if test_marketplace_connection "localhost:${port}"; then
            FOUND_ENDPOINT=1
            break
        fi
    done
fi

if [ $FOUND_ENDPOINT -eq 0 ]; then
    echo "ERROR: Could not connect to marketplace service"
    echo "Please ensure the marketplace service is running"
    echo "Tried:"
    echo "- Original URL: $ORIGINAL_URL"
    echo "- Alternative ports on localhost: 9000, 8082, 8080, 3000"
    exit 1
fi

echo "Using marketplace at: $MARKETPLACE_URL"

# Now try the models endpoint specifically
echo "Fetching models..."
MODELS_RESPONSE=$(curl -k -s -S "${MARKETPLACE_URL}/blockchain/models" 2>&1)
CURL_EXIT=$?

if [ $CURL_EXIT -ne 0 ]; then
    echo "ERROR: Failed to fetch models (exit code $CURL_EXIT)"
    echo "Response: $MODELS_RESPONSE"
    exit 1
fi

# Validate JSON response
if ! echo "$MODELS_RESPONSE" | jq empty 2>/dev/null; then
    echo "ERROR: Invalid JSON response"
    echo "Raw response:"
    echo "$MODELS_RESPONSE"
    exit 1
fi

# Count total models
TOTAL_MODELS=$(echo "$MODELS_RESPONSE" | jq '.models | length')
echo "Found $TOTAL_MODELS models, searching for matches..."

# Process models and store results with scores
echo "$MODELS_RESPONSE" | jq -r '.models[] | "\(.id)|\(.name)"' | while IFS="|" read -r id name; do
    if [ ! -z "$name" ]; then
        SIMILARITY=$(calculate_similarity "$MODEL_SEARCH" "$name")
        if (( $(echo "$SIMILARITY > 0.3" | bc -l) )); then
            echo "$SIMILARITY|$id|$name" >> "$TEMP_RESULTS"
        fi
    fi
done

# Check if we found any matches
if [ ! -s "$TEMP_RESULTS" ]; then
    echo "No matches found for '$MODEL_SEARCH'"
    exit 0
fi

# Sort and display results
echo -e "\nMatches found (sorted by relevance):"
echo "----------------------------------------"
sort -r -t'|' -k1,1 "$TEMP_RESULTS" | while IFS="|" read -r score id name; do
    printf "Match Score: %.2f\n" "$score"
    printf "Model ID: %s\n" "$id"
    printf "Name: %s\n" "$name"
    echo "----------------------------------------"
done

echo -e "\nTo use a model, set the MODEL_ID environment variable:"
echo "export MODEL_ID=<model-id>"
