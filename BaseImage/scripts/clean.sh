
#!/bin/bash
set -euo pipefail

echo "Stopping all running containers..."
docker stop $(docker ps -aq) 2>/dev/null || true

echo "Removing all containers..."
docker rm $(docker ps -aq) 2>/dev/null || true

echo "Removing all images..."
docker rmi $(docker images -q) 2>/dev/null || true

echo "Removing all volumes..."
docker volume rm $(docker volume ls -q) 2>/dev/null || true

echo "Removing all networks..."
docker network rm $(docker network ls -q) 2>/dev/null || true

echo "Pruning the system..."
docker system prune -af --volumes

# Clean up local directories if they exist
if [ -d "morpheus-node" ]; then
    echo "Removing morpheus-node directory..."
    rm -rf morpheus-node
fi

if [ -d "data" ]; then
    echo "Removing data directory..."
    rm -rf data
fi

echo "Docker environment has been completely cleaned."
echo "You can now run your setup scripts for a fresh installation."