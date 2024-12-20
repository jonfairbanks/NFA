#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${SCRIPT_DIR}/utils/env_loader.sh"
source "${SCRIPT_DIR}/utils/docker_helper.sh"

# Ensure base .env exists
create_base_env "${SCRIPT_DIR}/.."

# Load environment variables
load_env || {
    echo "Error: Failed to load environment variables"
    exit 1
}

# Check for required .env file before proceeding
if [ ! -f "${SCRIPT_DIR}/../.env" ]; then
    echo "Error: No .env file found in BaseImage project root"
    exit 1
fi

# Enable strict modes for safer scripting
set -euo pipefail
IFS=$'\n\t'

# Redirect all output to a log file and the console
exec > >(tee -i setup.log)
exec 2>&1

# Cleanup function
cleanup() {
  echo "Cleaning up..."
  # Add any necessary cleanup commands here
}
trap cleanup EXIT

# Prompt for sudo password upfront
sudo -v

# Keep sudo alive
while true; do
  sudo -n true
  sleep 60
  kill -0 "$$" || exit
done 2>/dev/null &

# Explicitly set PATH to include Homebrew directories
# Common Homebrew installation paths:
# - Intel Macs: /usr/local/bin
# - Apple Silicon Macs: /opt/homebrew/bin
if [ -d "/opt/homebrew/bin" ]; then
  export PATH="/opt/homebrew/bin:$PATH"
elif [ -d "/usr/local/bin" ]; then
  export PATH="/usr/local/bin:$PATH"
else
  echo "Homebrew installation directory not found. Please install Homebrew or update the script with the correct path."
  exit 1
fi

# Verify that brew is accessible
if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew is not installed or not found in PATH."
  exit 1
fi

# Determine package manager and set commands as arrays
if command -v apt-get >/dev/null 2>&1; then
  PACKAGE_MANAGER="apt-get"
  INSTALL_CMD=(sudo apt-get install -y)
  UPDATE_CMD=(sudo apt-get update)
elif command -v yum >/dev/null 2>&1; then
  PACKAGE_MANAGER="yum"
  INSTALL_CMD=(sudo yum install -y)
  UPDATE_CMD=(sudo yum check-update)
elif command -v brew >/dev/null 2>&1; then
  PACKAGE_MANAGER="brew"
  INSTALL_CMD=(brew install)
  UPDATE_CMD=(brew update)
else
  echo "Unsupported package manager. Please install dependencies manually."
  exit 1
fi

# Validate repository URL
DEFAULT_PROXY_ROUTER_PATH="https://github.com/Lumerin-protocol/Morpheus-Lumerin-Node.git"
if [[ ! "$DEFAULT_PROXY_ROUTER_PATH" =~ ^https://github\.com/ ]]; then
  echo "Invalid repository URL: $DEFAULT_PROXY_ROUTER_PATH" >&2
  exit 1
fi

# ==================== NEW SECTION: Load Execution Directory .env ====================
# Load environment variables
load_env || {
    echo "Error: No .env file found in current or parent directories"
    exit 1
}

# Verify required environment variables
validate_env "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" || exit 1
# ==================== END: Load Execution Directory .env ====================

# Default configurations
DEFAULT_PORT=9000  # Changed from 8080 to 9000
DEFAULT_CLONE_DIR="morpheus-node"
CLONE_DIR=$DEFAULT_CLONE_DIR

# Function to print usage information
usage() {
  echo "Usage: $0 [-p port] [-e key=value] [-d clone_dir]..."
  echo "  -p  Set the port for the HTTP API (default: $DEFAULT_PORT)"
  echo "  -e  Provide additional environment variables in key=value format"
  echo "  -d  Specify the clone directory (default: $DEFAULT_CLONE_DIR)"
  echo "  -h  Display this help message"
}

# Parse command line arguments
PORT=$DEFAULT_PORT
EXTRA_ENV_VARS=()  # Initialize EXTRA_ENV_VARS as an empty array
while getopts "p:e:d:h" opt; do
  case $opt in
    p)
      PORT=$OPTARG
      ;;
    e)
      EXTRA_ENV_VARS+=("$OPTARG")
      ;;
    d)
      CLONE_DIR=$OPTARG
      ;;
    h)
      usage
      exit 0
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      usage
      exit 1
      ;;
  esac
done

# Export extra environment variables if any are provided
if [ ${#EXTRA_ENV_VARS[@]} -gt 0 ]; then
  for VAR in "${EXTRA_ENV_VARS[@]}"; do
    if [ -z "$VAR" ]; then
      continue  # Skip uninitialized or empty variables
    fi
    if [[ $VAR =~ ^[A-Za-z_][A-Za-z0-9_]*=.+$ ]]; then
      export "$VAR"
    else
      echo "Invalid environment variable format: $VAR" >&2
      exit 1
    fi
  done
fi

# Install dependencies if not already installed
install_dependencies() {
  echo "Installing dependencies..."
  
  # Execute the update command
  "${UPDATE_CMD[@]}" || { echo "Package manager update failed"; exit 1; }

  # Install git
  if ! command -v git >/dev/null 2>&1; then
    echo "Installing git..."
    "${INSTALL_CMD[@]}" git || { echo "Failed to install git"; exit 1; }
  else
    echo "git is already installed."
  fi

  # Install Docker
  if ! command -v docker >/dev/null 2>&1; then
    echo "Installing Docker..."
    if [ "$PACKAGE_MANAGER" = "brew" ]; then
      "${INSTALL_CMD[@]}" --cask docker || { echo "Failed to install Docker"; exit 1; }
      open /Applications/Docker.app
      echo "Docker installed. Please ensure Docker Desktop is running."
    else
      "${INSTALL_CMD[@]}" docker.io || { echo "Failed to install Docker"; exit 1; }
      sudo systemctl start docker
      sudo systemctl enable docker
      sudo usermod -aG docker "$USER"
      echo "Added $USER to the docker group. Please log out and log back in to apply the changes."
    fi
  else
    echo "Docker is already installed."
  fi

  # Install Docker Compose
  if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
    echo "Installing Docker with Compose V2..."
    if [ "$PACKAGE_MANAGER" = "brew" ]; then
      "${INSTALL_CMD[@]}" --cask docker || { echo "Failed to install Docker"; exit 1; }
      open /Applications/Docker.app
      echo "Docker installed. Please ensure Docker Desktop is running."
    else
      "${INSTALL_CMD[@]}" docker.io || { echo "Failed to install Docker"; exit 1; }
      sudo systemctl start docker
      sudo systemctl enable docker
      sudo usermod -aG docker "$USER"
      echo "Added $USER to the docker group. Please log out and log back in to apply the changes."
    fi
  else
    echo "Docker with Compose V2 is already installed."
  fi
}

install_dependencies

# Detect host architecture and set platform
HOST_ARCH=$(uname -m)
case "${HOST_ARCH}" in
    x86_64)  PLATFORM="linux/amd64" ;;
    aarch64|arm64) PLATFORM="linux/arm64" ;;
    *)       
        echo "Unsupported architecture: ${HOST_ARCH}"
        exit 1
        ;;
esac

# Clone the proxy-router project
if [ ! -d "$CLONE_DIR" ]; then
  echo "Cloning the proxy-router project to $CLONE_DIR..."
  git clone "$DEFAULT_PROXY_ROUTER_PATH" "$CLONE_DIR" || { echo "Git clone failed"; exit 1; }
else
  echo "Directory $CLONE_DIR already exists. Skipping clone."
fi

# Navigate to the project directory
cd "$CLONE_DIR/proxy-router" || { echo "Failed to navigate to the project directory."; exit 1; }

# Setup config files from examples
echo "Setting up configuration files..."

# Setup .env file from example
if [ ! -f ".env" ]; then
    if [ ! -f ".env.example" ]; then
        echo "Error: .env.example not found in proxy-router directory"
        exit 1
    fi
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

# Setup models-config.json from example
if [ ! -f "models-config.json" ]; then
    if [ ! -f "models-config.json.example" ]; then
        echo "Error: models-config.json.example not found"
        exit 1
    fi
    echo "Creating models-config.json from example..."
    cp models-config.json.example models-config.json
fi

# Update the required variables in .env
set_env_var ".env" "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WEB_ADDRESS" "0.0.0.0:${PORT}"
set_env_var ".env" "WEB_PUBLIC_URL" "http://localhost:${PORT}"

# Verify config files exist and have content
if [ ! -f .env ] || [ ! -s .env ]; then
    echo "Error: .env file is missing or empty"
    exit 1
fi

if [ ! -f models-config.json ] || [ ! -s models-config.json ]; then
    echo "Error: models-config.json file is missing or empty"
    exit 1
fi

echo "Configuration files are ready."

# Setup .env file from example if it doesn't exist
if [ ! -f ".env" ] && [ -f ".env.example" ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
elif [ ! -f ".env" ]; then
    echo "Error: Neither .env nor .env.example found in proxy-router directory"
    exit 1
fi

# Update the required variables in .env
set_env_var ".env" "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WALLET_PRIVATE_KEY" "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
set_env_var ".env" "WEB_ADDRESS" "0.0.0.0:${PORT}"
set_env_var ".env" "WEB_PUBLIC_URL" "http://localhost:${PORT}"

# Setup .env file from example if it doesn't exist
if [ ! -f ".env" ] && [ -f ".env.example" ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
elif [ ! -f ".env" ]; then
    echo "Error: Neither .env nor .env.example found in proxy-router directory"
    exit 1
fi

# Update environment variables in .env
cat > .env << EOL
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
WALLET_PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
DIAMOND_CONTRACT_ADDRESS=0x10777866547c53cbd69b02c5c76369d7e24e7b10
MOR_TOKEN_ADDRESS=0x34a285a1b1c166420df5b6630132542923b5b27e
EXPLORER_API_URL="https://api-sepolia.arbiscan.io/api"
ETH_NODE_CHAIN_ID=421614
ETH_NODE_USE_SUBSCRIPTIONS=false
ETH_NODE_ADDRESS="https://arbitrum-sepolia.blockpi.network/v1/rpc/public"
ENVIRONMENT=development
ETH_NODE_LEGACY_TX=false
OPENAI_BASE_URL="http://localhost:8080/v1"
OPENAI_KEY="not-needed"
PROXY_STORE_CHAT_CONTEXT=true
PROXY_STORAGE_PATH=/app/data/
LOG_COLOR=true
WEB_ADDRESS=0.0.0.0:${PORT}
WEB_PUBLIC_URL=http://localhost:${PORT}
AI_ENGINE_OPENAI_BASE_URL="http://localhost:8080/v1"
EOL

# Verify .env file was created and has content
if [ ! -f .env ] || [ ! -s .env ]; then
    echo "Error: .env file is missing or empty"
    exit 1
fi

echo "Environment file contents:"
cat .env

# Set up models-config.json
# Set up models-config.json first (before .env setup)
if [ ! -f "models-config.json" ] && [ -f "models-config.json.example" ]; then
    echo "Creating models-config.json from example..."
    cp models-config.json.example models-config.json
elif [ ! -f "models-config.json" ]; then
    echo "Error: Neither models-config.json nor models-config.json.example found"
    exit 1
fi

# === BEGIN: Setup .env File ===
# Create initial .env file with required values
cat > "$CLONE_DIR/proxy-router/.env" << EOL
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
WALLET_PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"
DIAMOND_CONTRACT_ADDRESS=0x10777866547c53cbd69b02c5c76369d7e24e7b10
MOR_TOKEN_ADDRESS=0x34a285a1b1c166420df5b6630132542923b5b27e
EXPLORER_API_URL="https://api-sepolia.arbiscan.io/api"
ETH_NODE_CHAIN_ID=421614
ETH_NODE_USE_SUBSCRIPTIONS=false
ETH_NODE_ADDRESS="https://arbitrum-sepolia.blockpi.network/v1/rpc/public"
ENVIRONMENT=development
ETH_NODE_LEGACY_TX=false
OPENAI_BASE_URL="http://localhost:8080/v1"
OPENAI_KEY="not-needed"
PROXY_STORE_CHAT_CONTEXT=true
PROXY_STORAGE_PATH=/app/data/
LOG_COLOR=true
WEB_ADDRESS=0.0.0.0:${PORT}
WEB_PUBLIC_URL=http://localhost:${PORT}
AI_ENGINE_OPENAI_BASE_URL="http://localhost:8080/v1"
EOL

# Verify .env file was created
if [ ! -f "$CLONE_DIR/proxy-router/.env" ]; then
    echo "Error: Failed to create .env file"
    exit 1
fi

echo "Created .env file:"
cat "$CLONE_DIR/proxy-router/.env"
# === END: Setup .env File ===

# ==================== NEW SECTION: Set MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY in proxy-router/.env ====================
# Retrieve the value from the execution directory's .env
PRIVATE_KEY="${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}"

# Function to set or update a variable in .env
set_env_var() {
  local env_file=$1
  local var_name=$2
  local var_value=$3

  if grep -q "^${var_name}=" "$env_file"; then
    # Variable exists, replace the line
    # Using sed compatible with macOS
    sed -i '' "s/^${var_name}=.*/${var_name}=\"${var_value}\"/" "$env_file"
    echo "Updated ${var_name} in ${env_file}."
  else
    # Variable does not exist, append to the file
    echo "${var_name}=\"${var_value}\"" >> "$env_file"
    echo "Added ${var_name} to ${env_file}."
  fi
}

# Set the MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY in proxy-router/.env
echo "Setting MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY in proxy-router/.env..."
set_env_var ".env" "MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY" "$PRIVATE_KEY"
# ==================== END: Set MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY in proxy-router/.env ====================

# Before building Docker image, ensure .env has required values
echo "Configuring environment..."
cat > .env << EOL
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="${PRIVATE_KEY}"
WALLET_PRIVATE_KEY="${PRIVATE_KEY}"
WEB_PUBLIC_URL="http://localhost:${PORT}"
WEB_PORT="${PORT}"
EOL

# Verify .env file was created
if [ ! -f .env ]; then
    echo "Error: Failed to create .env file"
    exit 1
fi

echo "Environment file contents:"
cat .env

# Create models-config.json from example if it doesn't exist
if [ ! -f "models-config.json" ]; then
    if [ ! -f "models-config.json.example" ]; then
        echo "Error: models-config.json.example not found"
        exit 1
    fi
    echo "Creating models-config.json from example..."
    cp models-config.json.example models-config.json
fi

# Verify models-config.json exists
if [ ! -f "models-config.json" ]; then
    echo "Error: Failed to create models-config.json"
    exit 1
fi

# Ensure data directory exists with correct permissions
mkdir -p data
chmod 777 data

# Stop any existing containers
docker compose down 2>/dev/null || true

# Build and start with proper platform and environment
DOCKER_DEFAULT_PLATFORM="${PLATFORM}" \
HTTP_PORT="${PORT}" \
MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY="${PRIVATE_KEY}" \
docker compose up -d --build

# More reliable health check
echo "Waiting for service to be healthy..."
attempt=0
max_attempts=30
until curl -s "http://localhost:${PORT}/health" >/dev/null 2>&1 || [ $attempt -eq $max_attempts ]; do
    attempt=$((attempt + 1))
    echo "Attempt $attempt/$max_attempts: Waiting for service..."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "Service failed to start. Logs:"
    docker compose logs
    exit 1
fi

echo "Service is running!"
echo "API endpoint: http://localhost:${PORT}"
echo "To view logs: docker compose logs -f"

# Provide status information
echo "Proxy Router is running and the HTTP API is exposed on port $PORT"
echo "You can access the API at http://localhost:$PORT"

# End of script

# Navigate to the project directory
cd "$CLONE_DIR/proxy-router" || { echo "Failed to navigate to the project directory."; exit 1; }

# Ensure proxy-router env exists before proceeding
cd "$CLONE_DIR/proxy-router" || { echo "Failed to navigate to the project directory."; exit 1; }
ensure_env_exists ".env" || exit 1

# Setup .env file from example if it doesn't exist
if [ ! -f ".env" ]; then
    if [ ! -f ".env.example" ]; then
        echo "Error: .env.example not found in proxy-router directory"
        exit 1
    fi
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

# Ensure wallet private key is set from BaseImage's .env
if [ -z "${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY:-}" ]; then
    echo "Error: MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY not found in BaseImage's .env"
    exit 1
fi

# Update wallet-related environment variables
sed -i.bak "s|^MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY=.*|MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY=${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}|" .env
sed -i.bak "s|^WALLET_PRIVATE_KEY=.*|WALLET_PRIVATE_KEY=${MARKETPLACE_TEST_NODE_WALLET_PRIVATE_KEY}|" .env

# Update port settings
sed -i.bak "s|^WEB_ADDRESS=.*|WEB_ADDRESS=0.0.0.0:${PORT}|" .env
sed -i.bak "s|^WEB_PUBLIC_URL=.*|WEB_PUBLIC_URL=http://localhost:${PORT}|" .env

# Remove backup files
rm -f .env.bak