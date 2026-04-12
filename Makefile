.PHONY: all build run test lint clean proto migrate

# Default: build both binaries
all: build

# Build Go binaries
build:
	go build -o bin/veltrix-api ./cmd/api
	go build -o bin/veltrix-agent ./cmd/agent

# Run the full stack via docker-compose
run:
	docker-compose up --build

# Run the API server locally (requires Postgres and Redis running)
run-api:
	go run ./cmd/api

# Run the agent locally
run-agent:
	go run ./cmd/agent

# Run tests
test:
	go test ./... -v -race

# Lint
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Generate protobuf code
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

# Apply database migrations (requires psql)
migrate:
	psql "$(VELTRIX_POSTGRES_URL)" -f migrations/001_init.sql
