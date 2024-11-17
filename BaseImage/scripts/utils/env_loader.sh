#!/bin/bash

# Function to find and load .env file from current or parent directories
load_env() {
    local current_dir="$PWD"
    local env_file=""
    local found_files=()
    
    # Look for .env in current directory and up to 3 parent directories
    for i in {0..3}; do
        local check_path="${current_dir}/.env"
        if [ -f "$check_path" ]; then
            found_files+=("$check_path")
        fi
        current_dir="$(dirname "$current_dir")"
    done

    # Load files in reverse order (parent to child)
    for ((i=${#found_files[@]}-1; i>=0; i--)); do
        env_file="${found_files[$i]}"
        echo "Loading environment variables from $env_file"
        set -a
        source "$env_file"
        set +a
    done

    [ ${#found_files[@]} -gt 0 ] && return 0 || return 1
}

# Function to ensure required variables are set
validate_env() {
    local required_vars=("$@")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var:-}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        echo "Error: Missing required environment variables:"
        printf '%s\n' "${missing_vars[@]}"
        return 1
    fi
    return 0
}

# Function to set or update a variable in .env
set_env_var() {
    local env_file=$1
    local var_name=$2
    local var_value=$3

    # Use a different delimiter (|) for sed when dealing with URLs
    if grep -q "^${var_name}=" "$env_file"; then
        sed -i '' "s|^${var_name}=.*|${var_name}=\"${var_value}\"|" "$env_file"
        echo "Updated ${var_name} in ${env_file}."
    else
        echo "${var_name}=\"${var_value}\"" >> "$env_file"
        echo "Added ${var_name} to ${env_file}."
    fi
}

# Function to check if .env file exists
ensure_env_exists() {
    local env_file=$1
    local example_file="${env_file}.example"
    
    if [ ! -f "$env_file" ]; then
        if [ -f "$example_file" ]; then
            echo "Creating $env_file from example..."
            cp "$example_file" "$env_file"
        else
            echo "Error: Neither $env_file nor $example_file found"
            return 1
        fi
    fi
    return 0
}

# Function to create base .env file if it doesn't exist
create_base_env() {
    local base_dir=$1
    local env_file="${base_dir}/.env"
    
    if [ ! -f "$env_file" ]; then
        echo "Creating base .env file..."
        cat > "$env_file" << EOL
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="6f3453bc036158833281b9e0204975bad0f2a4407c4ab32132bb0f9e6e692cbb"
# Add other base environment variables as needed
EOL
        echo "Created base .env file at $env_file"
    fi
}