
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

# Check if a provider name/id was provided
if [ -z "$1" ]; then
    echo "Listing all available providers..."
    LIST_ALL=true
else
    LIST_ALL=false
    PROVIDER_SEARCH="$1"
fi

check_dependencies

MARKETPLACE_PORT=9000
echo "Starting marketplace service check..."

PROVIDERS_URL="http://localhost:${MARKETPLACE_PORT}/blockchain/providers"
echo "Testing marketplace API at $PROVIDERS_URL"

# Get the providers
PROVIDERS_RESPONSE=$(curl -s \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    "${PROVIDERS_URL}?limit=100&order=desc" 2>/dev/null)

CURL_EXIT=$?

if [ $CURL_EXIT -ne 0 ]; then
    echo "Error accessing providers endpoint (exit code: $CURL_EXIT)"
    echo "Failed URL: $PROVIDERS_URL"
    curl -v "$PROVIDERS_URL"
    exit 1
fi

# Create temporary file for storing results
TEMP_RESULTS=$(mktemp)
trap 'rm -f $TEMP_RESULTS' EXIT

# Parse and validate JSON response
if ! echo "$PROVIDERS_RESPONSE" | jq -e . >/dev/null 2>&1; then
    echo "Error: Invalid JSON response"
    echo "Raw response:"
    echo "$PROVIDERS_RESPONSE"
    exit 1
fi

# Process providers with fuzzy matching on both ID and Name
if [ $LIST_ALL = false ]; then
    echo "$PROVIDERS_RESPONSE" | jq -r '.providers[] | "\(.Id)|\(.Name)"' | while IFS="|" read -r id name; do
        if [ ! -z "$name" ]; then
            # Check similarity with both ID and Name
            ID_SIMILARITY=$(calculate_similarity "$PROVIDER_SEARCH" "$id")
            NAME_SIMILARITY=$(calculate_similarity "$PROVIDER_SEARCH" "$name")
            # Use the higher similarity score
            if (( $(echo "$ID_SIMILARITY > $NAME_SIMILARITY" | bc -l) )); then
                SIMILARITY=$ID_SIMILARITY
            else
                SIMILARITY=$NAME_SIMILARITY
            fi
            if (( $(echo "$SIMILARITY > 0.3" | bc -l) )); then
                echo "$SIMILARITY|$id|$name" >> "$TEMP_RESULTS"
            fi
        fi
    done
fi

# Display results
if [ $LIST_ALL = true ]; then
    echo -e "\nAvailable providers:"
    # Get headers from the first provider's keys
    HEADERS=$(echo "$PROVIDERS_RESPONSE" | jq -r '.providers[0] | keys_unsorted[]')
    
    # Print header row
    printf "\033[1m"  # Bold
    echo "$HEADERS" | tr '\n' '|' | awk -F'|' '{
        for(i=1; i<=NF; i++) {
            printf "%-30s ", $i
        }
        printf "\n"
    }'
    printf "\033[0m"
    
    # Print separator
    HEADER_COUNT=$(echo "$HEADERS" | wc -l)
    printf "%0.s-" $(seq 1 $((HEADER_COUNT * 31)))
    printf "\n"
    
    # Print each provider's data
    echo "$PROVIDERS_RESPONSE" | jq -r '.providers[] | [.[]|tostring] | join("|")' | while IFS="|" read -r fields; do
        echo "$fields" | tr '|' '\n' | awk '{
            printf "%-30s ", $0
        }'
        printf "\n"
    done
else
    if [ ! -s "$TEMP_RESULTS" ]; then
        echo "No matches found for '$PROVIDER_SEARCH'"
    else
        echo -e "\nMatches found (sorted by similarity):"
        
        # Print header with similarity score first
        printf "\033[1m%-15s " "MATCH SCORE"
        HEADERS=$(echo "$PROVIDERS_RESPONSE" | jq -r '.providers[0] | keys_unsorted[]')
        echo "$HEADERS" | tr '\n' '|' | awk -F'|' '{
            for(i=1; i<=NF; i++) {
                printf "%-30s ", $i
            }
            printf "\n"
        }'
        printf "\033[0m"
        
        # Print separator
        HEADER_COUNT=$(echo "$HEADERS" | wc -l)
        printf "%0.s-" $(seq 1 $((HEADER_COUNT * 31 + 15)))
        printf "\n"
        
        # Sort and display results with all fields
        sort -nr -t'|' -k1,1 "$TEMP_RESULTS" | while IFS="|" read -r score id name; do
            printf "%-15.4f " "$score"
            echo "$PROVIDERS_RESPONSE" | jq -r ".providers[] | select(.Id == \"$id\") | [.[]|tostring] | join(\"|\")" | tr '|' '\n' | awk '{
                printf "%-30s ", $0
            }'
            printf "\n"
        done
        
        echo -e "\nTo use a provider, add to your .env file:"
        echo "PROVIDER_ID=<provider-id>"
    fi
fi

exit 0