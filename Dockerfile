# Dockerfile
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o trawler-app .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create new user and group
RUN addgroup -S appgroup && adduser -S -G appgroup appuser

# Create data directories and set permissions
RUN mkdir -p /data/certs/online \
    mkdir -p /data/certs/offline \
    mkdir -p /data/crls/online \
    mkdir -p /data/crls/offline \
    mkdir -p /data/git && \
    chown -R appuser:appgroup /data
# Change owner of group
RUN mkdir /trawler && chown -R appuser:appgroup /trawler

# Switch to appuser
USER appuser

WORKDIR /trawler/

# Copy binary from builder
COPY --from=builder /app/trawler-app .

# Run the application
CMD ["./trawler-app"]
