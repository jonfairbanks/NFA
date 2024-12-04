#!/bin/zsh

#set -u  # Error on undefined variables

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

# Check if a model name was provided - modified to make it optional
if [ -z "$1" ]; then
    echo "Listing all available models..."
    LIST_ALL=true
else
    LIST_ALL=false
    MODEL_SEARCH="$1"
fi

check_dependencies

# Only check marketplace port - don't try others
MARKETPLACE_PORT=9000
echo "Starting marketplace service check..."

# Fix: Use the correct blockchain API endpoint for models
MODELS_URL="http://localhost:${MARKETPLACE_PORT}/blockchain/models"
echo "Testing marketplace API at $MODELS_URL"

# Get the models using the new API format with pagination params
MODELS_RESPONSE=$(curl -s \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    "${MODELS_URL}?limit=100&order=desc" 2>/dev/null)

CURL_EXIT=$?

if [ $CURL_EXIT -ne 0 ]; then
    echo "Error accessing models endpoint (exit code: $CURL_EXIT)"
    echo "Failed URL: $MODELS_URL"
    curl -v "$MODELS_URL"
    exit 1
fi

# Create temporary file for storing results
TEMP_RESULTS=$(mktemp)
trap 'rm -f $TEMP_RESULTS' EXIT

# Parse and validate JSON response
if ! echo "$MODELS_RESPONSE" | jq -e . >/dev/null 2>&1; then
    echo "Error: Invalid JSON response"
    echo "Raw response:"
    echo "$MODELS_RESPONSE"
    exit 1
fi

# Process models with updated JSON structure
echo "$MODELS_RESPONSE" | jq -r '.models[] | "\(.Id)|\(.Name)"' | while IFS="|" read -r id name; do
    if [ ! -z "$name" ]; then
        SIMILARITY=$(calculate_similarity "$MODEL_SEARCH" "$name")
        if (( $(echo "$SIMILARITY > 0.3" | bc -l) )); then
            echo "$SIMILARITY|$id|$name" >> "$TEMP_RESULTS"
        fi
    fi
done

# Modified results display section
if [ $LIST_ALL = true ]; then
    echo -e "\nAvailable models:"
    printf "\033[1m%-66s %-30s\033[0m\n" "MODEL ID" "NAME"
    printf "%0.s-" {1..96}
    printf "\n"
    
    # Display all models without similarity scoring
    echo "$MODELS_RESPONSE" | jq -r '.models[] | "\(.Id)|\(.Name)"' | while IFS="|" read -r id name; do
        printf "%-66s %-30s\n" "$id" "$name"
    done
else
    # Display results with better formatting
    if [ ! -s "$TEMP_RESULTS" ]; then
        echo "No matches found for '$MODEL_SEARCH'"
    else
        echo -e "\nMatches found (sorted by similarity):"
        printf "\033[1m%-15s %-66s %-30s\033[0m\n" "MATCH SCORE" "MODEL ID" "NAME"
        printf "%0.s-" {1..111}
        printf "\n"
        
        # Sort results by similarity score
        sort -nr -t'|' -k1,1 "$TEMP_RESULTS" | while IFS="|" read -r score id name; do
            printf "%-15.4f %-66s %-30s\n" "$score" "$id" "$name"
        done
        
        echo -e "\nTo use a model, add to your .env file:"
        echo "MODEL_ID=<model-id>"
    fi
fi

exit 0
