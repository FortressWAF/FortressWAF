# ============================================================================
# FortressWAF Proxy - Multi-stage Docker Build
# ============================================================================
# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Cache Go module downloads
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build version metadata
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.Version=${VERSION}' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.Commit=${COMMIT}' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.BuildDate=${BUILD_DATE}'" \
    -o /app/fortress-proxy \
    ./cmd/proxy

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.Version=${VERSION}' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.Commit=${COMMIT}' \
      -X 'github.com/FortressWAF/FortressWAF/internal/version.BuildDate=${BUILD_DATE}'" \
    -o /app/fortressctl \
    ./cmd/ctl

# ============================================================================
# Stage 2: Final runtime image (distroless)
FROM gcr.io/distroless/static-debian12:latest

LABEL maintainer="FortressWAF Team"
LABEL org.opencontainers.image.title="FortressWAF Proxy"
LABEL org.opencontainers.image.description="Enterprise Web Application Firewall with ML-powered threat detection"
LABEL org.opencontainers.image.source="https://github.com/FortressWAF/FortressWAF"

# Run as non-root
USER 65534:65534

COPY --from=builder /app/fortress-proxy /fortress-proxy
COPY --from=builder /app/fortressctl /fortressctl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy default rule files
COPY --chown=65534:65534 rules/ /etc/fortresswaf/rules/

ENV TZ=UTC

WORKDIR /

# distroless does not have wget/curl, so use the binary's built-in health endpoint
# The proxy exposes /health on port 8080
EXPOSE 80 443 8443 8080

ENTRYPOINT ["/fortress-proxy"]
