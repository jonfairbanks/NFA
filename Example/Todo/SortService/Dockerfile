# Use the official Golang image as a build stage
FROM golang:1.20-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN go build -o main .

# Use a minimal base image to run the application
FROM alpine:3.18

# Set the working directory inside the container
WORKDIR /app

# Copy the built application from the builder image
COPY --from=builder /app/main .

# Expose port 8080 for the application
EXPOSE 8080

# Command to run the application
CMD ["./main"]
