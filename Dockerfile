# Multi-stage build for WGO
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o wgo ./cmd/wgo

# Final stage - minimal image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    aws-cli \
    wget \
    unzip

# Install Terraform manually
RUN wget https://releases.hashicorp.com/terraform/1.7.5/terraform_1.7.5_linux_arm64.zip \
    && unzip terraform_1.7.5_linux_arm64.zip \
    && mv terraform /usr/local/bin/ \
    && rm terraform_1.7.5_linux_arm64.zip

# Create non-root user
RUN addgroup -g 1000 wgo && \
    adduser -u 1000 -G wgo -D wgo

# Copy binary from builder
COPY --from=builder /build/wgo /usr/local/bin/wgo

# Create working directory
WORKDIR /workspace

# Switch to non-root user
USER wgo

# Verify installation
RUN wgo version

# Default command
ENTRYPOINT ["wgo"]
CMD ["--help"]