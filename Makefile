.PHONY: all build gen clean server client proto deps

# Variables
SERVER_BINARY = bin/jacuzzi-server
CLIENT_BINARY = bin/jacuzzi-client
PROTO_DIR = proto
PROTO_GEN_DIR = proto/gen
DB_DIR = data/db

all: deps gen build

# Install dependencies
deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

# Generate protobuf code
gen: proto

proto:
	buf generate

# Build binaries
build: server client

server:
	mkdir -p bin
	go build -o $(SERVER_BINARY) ./server

client:
	mkdir -p bin
	go build -o $(CLIENT_BINARY) ./client

# Run server
run-server: server
	$(SERVER_BINARY)

# Run client
run-client: client
	$(CLIENT_BINARY)

# Clean generated files and binaries
clean:
	rm -rf $(PROTO_GEN_DIR)
	rm -rf bin/
	rm -rf $(DB_DIR)
	mkdir -p $(DB_DIR) bin/

# Development helpers
dev-server:
	go run ./server -db-type sqlite -db-name jacuzzi-dev.db

dev-client:
	go run ./client

# Format code
fmt:
	go fmt ./...
	buf format -w

# Run tests
test:
	go test -v ./...

# Lint code
lint:
	buf lint
	golangci-lint run