# Use the official Golang image
FROM golang:1.20-alpine

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o nfa-base .

# Expose the port
EXPOSE 8080

# Set environment variables (to be overridden at runtime)
ENV MARKETPLACE_URL=http://localhost:8082
ENV MODEL_ID=your_model_id
# Default to 1 hour
ENV SESSION_DURATION=1h  


# Run the application
CMD ["./nfa-base"]