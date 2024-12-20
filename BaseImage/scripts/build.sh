
#!/bin/sh

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the proxy router
echo "Building nfa-proxy..."
go build -o bin/nfa-proxy \
    -ldflags "-s -w" \
    -trimpath \
    ./main.go

# Make the binary executable
chmod +x bin/nfa-proxy

echo "Build complete!"