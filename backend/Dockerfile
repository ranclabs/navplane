# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy go mod files
COPY go.mod ./
# Uncomment when you have dependencies:
# COPY go.sum ./
# RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o navplane main.go

# Runtime stage
FROM scratch

WORKDIR /app

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/navplane .

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/navplane"]
