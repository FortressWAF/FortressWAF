# FortressWAF Vulnerable Test App
# WARNING: This is intentionally vulnerable. NEVER deploy to production.
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/vuln-app .

FROM gcr.io/distroless/static-debian12:latest
LABEL maintainer="FortressWAF Team"
LABEL org.opencontainers.image.title="FortressWAF VulnApp"
LABEL org.opencontainers.image.description="Deliberately vulnerable test app for FortressWAF testing"
USER 65534:65534
COPY --from=builder /app/vuln-app /vuln-app
EXPOSE 8080
ENV VULN_APP_PORT=8080
ENTRYPOINT ["/vuln-app"]
