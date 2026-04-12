# Multi-stage Dockerfile for Veltrix Go services.
#
# Builds two targets:
#   - api:   the control plane API server
#   - agent: the node agent
#
# Usage:
#   docker build --target api -t veltrix-api .
#   docker build --target agent -t veltrix-agent .
#
# Or via docker-compose (see docker-compose.yml).

# ---------------------------------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copy source
COPY . .

# Build API server
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/veltrix-api ./cmd/api

# Build agent
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/veltrix-agent ./cmd/agent

# ---------------------------------------------------------------------------
# Stage 2a: API server image
# ---------------------------------------------------------------------------
FROM alpine:3.19 AS api

RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/veltrix-api /usr/local/bin/veltrix-api

EXPOSE 8080
ENTRYPOINT ["veltrix-api"]

# ---------------------------------------------------------------------------
# Stage 2b: Agent image
# ---------------------------------------------------------------------------
FROM alpine:3.19 AS agent

RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/veltrix-agent /usr/local/bin/veltrix-agent

ENTRYPOINT ["veltrix-agent"]
