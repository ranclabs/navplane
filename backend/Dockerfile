# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o navplane ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates and wget for healthcheck
RUN apk --no-cache add ca-certificates wget

# Copy the binary
COPY --from=builder /app/navplane .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/navplane"]
