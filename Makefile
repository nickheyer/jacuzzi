.PHONY: all build gen clean server client proto deps

SERVER_BINARY = bin/jacuzzi-server
CLIENT_BINARY = bin/jacuzzi-client
SERVER_CMD = cmd/server/main.go
CLIENT_CMD = cmd/client/main.go
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

# Generate protobuf code + build server UI (svelte app)
gen: proto
	go generate ./...

proto:
	buf generate

# Build binaries
build: clean gen server client

server:
	mkdir -p bin
	go build -o $(SERVER_BINARY) $(SERVER_CMD)

client:
	mkdir -p bin
	go build -o $(CLIENT_BINARY) $(CLIENT_CMD)

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