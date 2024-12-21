# Getting Started with NFA Proxy for Agent Builders

This guide will help you set up and run the **NFA Proxy**, enabling you to build and test your agent with the proxy in place. The NFA Proxy acts as an intermediary between your agent and the Morpheus Marketplace, handling session initiation and request forwarding.

---

## Prerequisites

Ensure you have the following installed on your Mac machine:

- **Git**: Version control system.

    ```bash
    brew install git
    ```

- **Docker Desktop for Mac**: Containerization platform.

    [Download and install Docker Desktop](https://www.docker.com/products/docker-desktop).

---

## Configuration Requirements

### 1. Clone the BaseImage Repository

Begin by cloning the **BaseImage** repository to your local machine:

    ```bash
    git clone https://github.com/MORpheus-Software/NFA.git
    ```

### 2. Navigate to the BaseImage Directory

Change to the BaseImage directory:

    ```bash
    cd BaseImage
    ```

### 3. Set Up Environment Variables

The BaseImage project uses environment variables for configuration. Follow these steps:

1. Copy the example environment file:

    ```bash
    cp example.env .env
    ```

2. Edit the `.env` file and replace the placeholder values:

    ```dotenv
    # .env
    WALLET_PRIVATE_KEY=your_private_key_here
    WALLET_ADDRESS=your_wallet_address_here
    PORT=8080
    MARKETPLACE_URL=http://localhost:9000/v1/chat/completions
    SESSION_DURATION=1h
    MODEL_ID=your_model_id_here
    ```

**Important Notes:**
- Replace all placeholder values with your actual configuration
- Use **test credentials** only - never commit real private keys
- The `MODEL_ID` is required for the proxy to function correctly

---

## Steps to Run the NFA Proxy with Docker Compose

### 1. Build the Docker Image

The project includes a build script with several options:

    ```bash
    chmod +x scripts/docker-build.sh
    ./scripts/docker-build.sh -t nfa-proxy -f Dockerfile.proxy
    ```

Build script options:
- `-t`: Specify the image tag (default: nfa-base)
- `-f`: Specify the Dockerfile to use (default: Dockerfile.proxy)
- `-p`: Specify platform(s) to build for (default: host architecture)
- `-a`: Add build arguments in key=value format

Example with multiple options:
    ```bash
    ./scripts/docker-build.sh -t nfa-proxy -f Dockerfile.proxy -p linux/amd64 -p linux/arm64
    ```

### 2. Run the NFA Proxy Container with Docker Compose

Start the NFA Proxy container using Docker Compose:

    ```bash
    docker-compose up -d
    ```

This command runs the `nfa-proxy` service defined in the `docker-compose.yml` file in detached mode.

### 3. Verify the NFA Proxy is Running

Check that the container is running:

    ```bash
    docker-compose ps
    ```

You should see the `nfa-proxy` service in the list.

Test the health endpoint:

    ```bash
    curl http://localhost:8080/health
    ```

You should receive a response indicating that the service is healthy.

---

## Usage Instructions

### Testing the Chat Endpoints

You can test the NFA Proxy by sending requests to the chat completion endpoint.

#### Example: Chat Completion Request

    ```bash
    curl -X POST http://localhost:8080/v1/chat/completions \
      -H "Content-Type: application/json" \
      -d '{
            "model": "YourModelName",
            "messages": [{"role": "user", "content": "Hello, agent!"}]
          }'
    ```

Replace `"YourModelName"` with the name of the model you are using.

#### Example: Streaming Chat Completion

    ```bash
    curl -N -X POST http://localhost:8080/v1/chat/completions \
      -H "Content-Type: application/json" \
      -d '{
            "model": "YourModelName",
            "messages": [{"role": "user", "content": "Stream this message."}],
            "stream": true
          }'
    ```

The `-N` flag keeps the connection open for streaming responses.

### Viewing Logs

To see the logs of the NFA Proxy container:

    ```bash
    docker-compose logs -f nfa-proxy
    ```

### Stopping the NFA Proxy

To stop and remove the container:

    ```bash
    docker-compose down
    ```

---

## Additional Notes

- **Environment Variables**: Ensure all required variables in the `.env` file are correctly set.
- **Port Configuration**: If port `8080` is in use, modify the `ports` section in the `docker-compose.yml` file to map to an available port.
- **Marketplace URL**: The `MARKETPLACE_URL` should point to a running instance of the marketplace. Adjust it if running the marketplace on a different host or port.

---

## Troubleshooting

- **Docker Not Running**: Ensure Docker Desktop is running before executing Docker commands.
- **Port Conflicts**: Check for port conflicts on `8080` and adjust accordingly.
- **Missing Dependencies**: Confirm that all prerequisites are installed and properly configured.
- **Environment Variable Issues**: Double-check the `.env` file for typos or incorrect values.

---

## Additional Scripts

To remove all containers, images, and volumes from your environment, run:
```bash
./scripts/clean.sh
```

To run the Morpheus Node locally for testing or development, run:
```bash
./scripts/runMorpheusNode.sh
```

---

You're now ready to build and test your agent with the NFA Proxy!