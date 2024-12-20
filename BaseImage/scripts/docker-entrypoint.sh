
#!/bin/sh
set -e

# Check if port is available
if ! nc -z localhost "${PORT:-8080}" 2>/dev/null; then
    echo "Port ${PORT:-8080} is available"
else
    echo "Warning: Port ${PORT:-8080} might be in use"
fi

exec "$@"