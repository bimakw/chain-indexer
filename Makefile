.PHONY: build run test lint clean docker-up docker-down migrate

# Build variables
BINARY_NAME=chain-indexer
BUILD_DIR=bin

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build the indexer binary
build:
	$(GOBUILD) -o $(BUILD_DIR)/indexer ./cmd/indexer
	$(GOBUILD) -o $(BUILD_DIR)/api ./cmd/api

# Run the indexer
run-indexer:
	$(GOBUILD) -o $(BUILD_DIR)/indexer ./cmd/indexer && ./$(BUILD_DIR)/indexer

# Run the API server
run-api:
	$(GOBUILD) -o $(BUILD_DIR)/api ./cmd/api && ./$(BUILD_DIR)/api

# Run tests
test:
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Run database migrations
migrate-up:
	migrate -path migrations -database "postgres://indexer:indexer@localhost:5432/chain_indexer?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://indexer:indexer@localhost:5432/chain_indexer?sslmode=disable" down

# Start Anvil with mainnet fork
anvil:
	anvil --fork-url https://eth.llamarpc.com --fork-block-number 19000000

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build indexer and API binaries"
	@echo "  run-indexer    - Build and run the indexer"
	@echo "  run-api        - Build and run the API server"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback database migrations"
	@echo "  anvil          - Start Anvil with mainnet fork"
