# Build stage
FROM golang:1.26.1-alpine3.23 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /app

# Install tools and data needed for HTTPS, time handling, and git-based version fallback
RUN apk --no-cache add ca-certificates tzdata git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary. If VERSION is not overridden, derive it from git like local builds do.
RUN RESOLVED_VERSION="${VERSION}"; \
    if [ "${RESOLVED_VERSION}" = "dev" ] && [ -d .git ]; then \
      RESOLVED_VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"; \
    fi; \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags="-s -w -X github.com/soli0222/diary-cli/internal/cli.Version=${RESOLVED_VERSION}" \
      -o /diary-cli ./cmd/diary-cli

# Runtime stage
FROM scratch

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /diary-cli /diary-cli

# Run as non-root (UID 65534 is nobody)
USER 65534:65534

ENTRYPOINT ["/diary-cli"]
