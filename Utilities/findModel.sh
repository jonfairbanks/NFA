#!/bin/zsh

#set -u  # Error on undefined variables

# Load environment variables from .env file
load_env() {
    if [ -f .env ]; then
        while IFS='=' read -r key value; do
            # Skip comments and empty lines
            [[ $key =~ ^#.*$ || -z $key ]] && continue
            # Remove quotes and export the variable
            value=$(echo "$value" | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")
            export "$key=$value"
            echo "Loaded: $key"
        done < .env
    fi
}

# Load environment variables
load_env

# Validate required environment variables
if [ -z "$PROVIDER_URL" ]; then
    echo "Error: PROVIDER_URL is not set in .env file"
    echo "Please add PROVIDER_URL=<your-provider-url> to your .env file"
    exit 1
fi

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
# Convert both strings to lowercase before comparison
s1 = '${s1}'.lower()
s2 = '${s2}'.lower()
score = Levenshtein.ratio(s1, s2)
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
MODELS_URL="${PROVIDER_URL}/blockchain/models"
echo "Testing marketplace API at $MODELS_URL"

# Create temp files for curl output and logging
TEMP_DIR=$(mktemp -d)
TEMP_RESULTS="${TEMP_DIR}/results.txt"
CURL_LOG="${TEMP_DIR}/curl.log"
CURL_BODY="${TEMP_DIR}/response.json"
CURL_FORMAT="${TEMP_DIR}/curl-format.txt"

# Create curl format file for detailed timing
cat > "${CURL_FORMAT}" << 'EOF'
    time_namelookup:  %{time_namelookup}s\n
       time_connect:  %{time_connect}s\n
    time_appconnect:  %{time_appconnect}s\n
   time_pretransfer:  %{time_pretransfer}s\n
      time_redirect:  %{time_redirect}s\n
 time_starttransfer:  %{time_starttransfer}s\n
                    ----------\n
         time_total:  %{time_total}s\n
EOF

# Cleanup function
cleanup() {
    rm -rf "${TEMP_DIR}"
}
trap cleanup EXIT

# Get the models using the new API format with pagination params
echo "Making request to: ${MODELS_URL}?limit=100&order=desc"
echo "Capturing full response details..."

# Make the curl request with detailed logging
curl -w "@${CURL_FORMAT}" -v -s \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -o "${CURL_BODY}" \
    -D "${CURL_LOG}" \
    "${MODELS_URL}?limit=100&order=desc" 2>&1 | tee "${TEMP_DIR}/full_output.log"

CURL_EXIT=$?

# Print formatted response details
echo -e "\n===== CURL REQUEST DETAILS ====="
echo -e "\033[1;36mRequest URL:\033[0m"
echo "  ${MODELS_URL}?limit=100&order=desc"

echo -e "\n\033[1;36mRequest Headers:\033[0m"
grep -E "^>" "${TEMP_DIR}/full_output.log" | sed 's/> /  /'

echo -e "\n\033[1;36mResponse Status:\033[0m"
head -n1 "${CURL_LOG}" | sed 's/^/  /'

echo -e "\n\033[1;36mResponse Headers:\033[0m"
grep -E "^<" "${TEMP_DIR}/full_output.log" | sed 's/< /  /'

echo -e "\n\033[1;36mResponse Size:\033[0m"
echo "  $(wc -c < "${CURL_BODY}" | xargs) bytes"

echo -e "\n\033[1;36mTiming Details:\033[0m"
cat "${TEMP_DIR}/full_output.log" | grep -E "time_|total" | sed 's/^/  /'

if [ $CURL_EXIT -ne 0 ]; then
    echo -e "\n\033[1;31mError:\033[0m curl failed with exit code $CURL_EXIT"
    echo -e "\033[1;31mFull debug log:\033[0m"
    cat "${TEMP_DIR}/full_output.log"
    exit 1
fi

# Store and validate response
MODELS_RESPONSE=$(cat "${CURL_BODY}")

# Always show the raw response first
echo -e "\n\033[1;36mRaw Response:\033[0m"
echo "${MODELS_RESPONSE}"

# Check if it's valid JSON
if ! echo "$MODELS_RESPONSE" | jq -e . >/dev/null 2>&1; then
    echo -e "\n\033[1;31mError:\033[0m Invalid JSON response"
    echo -e "\033[1;31mParsed response attempt:\033[0m"
    echo "$MODELS_RESPONSE" | jq -C . || echo "Could not parse as JSON"
    exit 1
fi

# Check if response contains models array
if ! echo "$MODELS_RESPONSE" | jq -e '.models' >/dev/null 2>&1; then
    echo -e "\n\033[1;31mError:\033[0m Response does not contain 'models' array"
    echo -e "\033[1;31mValid JSON but incorrect structure:\033[0m"
    echo "$MODELS_RESPONSE" | jq -C .
    exit 1
fi

# Only continue if we have valid data
MODELS_COUNT=$(echo "$MODELS_RESPONSE" | jq '.models | length')
if [ "$MODELS_COUNT" -eq 0 ]; then
    echo -e "\n\033[1;33mWarning:\033[0m No models found in response"
    echo -e "Response structure is valid but empty."
    exit 0
fi

echo -e "\n\033[1;32mRequest successful!\033[0m"
echo "Found ${MODELS_COUNT} models"

# Optional: Display sample of response data
echo -e "\n\033[1;36mResponse Preview:\033[0m"
echo "$MODELS_RESPONSE" | jq -C '.models | length as $total | "Total models: \($total)\nFirst model:", .models[0]'

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
    # Get headers from the first model's keys
    HEADERS=$(echo "$MODELS_RESPONSE" | jq -r '.models[0] | keys_unsorted[]')
    
    # Print header row
    printf "\033[1m"  # Bold
    echo "$HEADERS" | tr '\n' '|' | awk -F'|' '{
        for(i=1; i<=NF; i++) {
            printf "%-30s ", $i
        }
        printf "\n"
    }'
    printf "\033[0m"  # Reset formatting
    
    # Print separator
    HEADER_COUNT=$(echo "$HEADERS" | wc -l)
    printf "%0.s-" $(seq 1 $((HEADER_COUNT * 31)))
    printf "\n"
    
    # Print each model's data
    echo "$MODELS_RESPONSE" | jq -r '.models[] | [.[]|tostring] | join("|")' | while IFS="|" read -r fields; do
        echo "$fields" | tr '|' '\n' | awk '{
            printf "%-30s ", $0
        }'
        printf "\n"
    done
else
    # Search results with all fields
    if [ ! -s "$TEMP_RESULTS" ]; then
        echo "No matches found for '$MODEL_SEARCH'"
    else
        echo -e "\nMatches found (sorted by similarity):"
        
        # Print header with similarity score first
        printf "\033[1m%-15s " "MATCH SCORE"
        HEADERS=$(echo "$MODELS_RESPONSE" | jq -r '.models[0] | keys_unsorted[]')
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
            echo "$MODELS_RESPONSE" | jq -r ".models[] | select(.Id == \"$id\") | [.[]|tostring] | join(\"|\")" | tr '|' '\n' | awk '{
                printf "%-30s ", $0
            }'
            printf "\n"
        done
        
        echo -e "\nTo use a model, add to your .env file:"
        echo "MODEL_ID=<model-id>"
    fi
fi

exit 0
