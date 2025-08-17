# Build stage
FROM golang:1.24.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o vn .

# Final stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/vn /vn

# Set the binary as entrypoint
ENTRYPOINT ["/vn"]

# Default command shows help
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="VN - Vulnerability Navigator"
LABEL org.opencontainers.image.description="A CLI tool for OWASP Top 10 security testing"
LABEL org.opencontainers.image.source="https://github.com/your-username/vn"
LABEL org.opencontainers.image.licenses="MIT" 