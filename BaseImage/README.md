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
    git clone https://github.com/YourOrganization/BaseImage.git
    ```

Replace `https://github.com/YourOrganization/BaseImage.git` with the actual repository URL if different.

### 2. Navigate to the BaseImage Directory

Change to the BaseImage directory:

    ```bash
    cd BaseImage
    ```

### 3. Set Up Environment Variables

Create a `.env` file in the [BaseImage](http://_vscodecontentref_/0) directory to configure necessary environment variables:

    ```bash
    touch .env
    ```

Edit the `.env` file and add the following content:

    ```dotenv
    # .env
    WALLET_PRIVATE_KEY=your_private_key_here
    WALLET_ADDRESS=your_wallet_address_here
    PORT=8080
    MARKETPLACE_URL=http://localhost:9000/v1/chat/completions
    SESSION_DURATION=1h
    ```

**Important:**

- Replace `your_private_key_here` with your test wallet private key.
- Replace `your_wallet_address_here` with your test wallet address.
- Use **test credentials** only. Do **not** use real private keys or sensitive data.

---

## Steps to Run the NFA Proxy in a Docker Container

### 1. Build the Docker Image

Use the provided build script to build the NFA Proxy Docker image:

    ```bash
    chmod +x scripts/docker-build.sh
    ./scripts/docker-build.sh -t nfa-proxy -f Dockerfile.proxy
    ```

This script builds the Docker image with the tag `nfa-proxy` using the `Dockerfile.proxy` file.

### 2. Run the NFA Proxy Container

Start the NFA Proxy container:

    ```bash
    docker run -d \
      --name nfa-proxy \
      --env-file .env \
      -p 8080:8080 \
      nfa-proxy
    ```

This command runs the `nfa-proxy` image in detached mode, names the container `nfa-proxy`, loads environment variables from the `.env` file, and maps port `8080` of the container to port `8080` on your machine.

### 3. Verify the NFA Proxy is Running

Check that the container is running:

    ```bash
    docker ps
    ```

You should see the `nfa-proxy` container in the list.

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
    docker logs -f nfa-proxy
    ```

### Stopping the NFA Proxy

To stop and remove the container:

    ```bash
    docker stop nfa-proxy
    docker rm nfa-proxy
    ```

---

## Additional Notes

- **Environment Variables**: Ensure all required variables in the `.env` file are correctly set.
- **Port Configuration**: If port `8080` is in use, modify the `-p` flag in the `docker run` command to map to an available port.
- **Marketplace URL**: The `MARKETPLACE_URL` should point to a running instance of the marketplace. Adjust it if running the marketplace on a different host or port.

---

## Troubleshooting

- **Docker Not Running**: Ensure Docker Desktop is running before executing Docker commands.
- **Port Conflicts**: Check for port conflicts on `8080` and adjust accordingly.
- **Missing Dependencies**: Confirm that all prerequisites are installed and properly configured.
- **Environment Variable Issues**: Double-check the `.env` file for typos or incorrect values.

---

You're now ready to build and test your agent with the NFA Proxy!