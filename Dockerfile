# Multi-stage build for VAINO - Finnish Creator God's Container
FROM golang:1.23-alpine AS builder

# Install dependencies for divine compilation
RUN apk add --no-cache git make

# Set working directory - Väinämoinen's workshop
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary - forge the divine tool
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o vaino ./cmd/vaino

# Final stage - minimal image for divine deployment
FROM alpine:3.19

# Install runtime dependencies for Finnish power
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    aws-cli \
    wget \
    unzip

# Install Terraform manually - tools for the creator god
RUN wget https://releases.hashicorp.com/terraform/1.7.5/terraform_1.7.5_linux_arm64.zip \
    && unzip terraform_1.7.5_linux_arm64.zip \
    && mv terraform /usr/local/bin/ \
    && rm terraform_1.7.5_linux_arm64.zip

# Create non-root user - Väinämoinen's servant
RUN addgroup -g 1000 vaino && \
    adduser -u 1000 -G vaino -D vaino

# Copy binary from builder - the divine artifact
COPY --from=builder /build/vaino /usr/local/bin/vaino

# Create working directory - sacred workspace
WORKDIR /workspace

# Switch to non-root user
USER vaino

# Verify installation - test the divine power
RUN vaino version

# Default command - invoke the Finnish creator god
ENTRYPOINT ["vaino"]
CMD ["--help"]