# Build stage
FROM golang:1.24.2-alpine AS builder

# Set the working directory
WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Final stage
FROM alpine:latest

# Add CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/app .

# Copy the private key file (if needed)
# Note: In production, consider using secrets or volume mounts instead
COPY --from=builder /app/*.p8 ./

# Expose the port your app runs on
EXPOSE 8080

# Run the application
CMD ["./app"]