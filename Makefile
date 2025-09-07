# Cosmos State Mesh Makefile

# Build variables
BINARY_NAME=state-mesh
BUILD_DIR=./bin
CMD_DIR=./cmd/state-mesh
DOCKER_IMAGE=cosmos/state-mesh
VERSION?=latest

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Database variables
DB_HOST?=localhost
DB_PORT?=5432
DB_NAME?=statemesh
DB_USER?=postgres
DB_PASSWORD?=password
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: all build clean test test-coverage test-integration deps docker-build docker-push help

# Default target
all: clean deps build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Generate GraphQL code
generate:
	@echo "Generating GraphQL code..."
	go run github.com/99designs/gqlgen generate

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Rolling back database migrations..."
	migrate -path ./migrations -database "$(DB_URL)" down

migrate-create:
	@echo "Creating new migration: $(name)"
	migrate create -ext sql -dir ./migrations $(name)

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 8081:8081 $(DOCKER_IMAGE):latest

# Development targets
dev-setup:
	@echo "Setting up development environment..."
	@make deps
	@make migrate-up
	@echo "Development environment ready"

dev-run:
	@echo "Starting development server..."
	$(BUILD_DIR)/$(BINARY_NAME) serve --config config.dev.yaml

dev-ingest:
	@echo "Starting development ingester..."
	$(BUILD_DIR)/$(BINARY_NAME) ingest --config config.dev.yaml

# Kubernetes/Helm targets
helm-install:
	@echo "Installing Helm chart..."
	helm install state-mesh ./deployments/helm/state-mesh

helm-upgrade:
	@echo "Upgrading Helm chart..."
	helm upgrade state-mesh ./deployments/helm/state-mesh

helm-uninstall:
	@echo "Uninstalling Helm chart..."
	helm uninstall state-mesh

# Monitoring targets
prometheus-up:
	@echo "Starting Prometheus..."
	docker run -d -p 9090:9090 -v $(PWD)/monitoring/prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus

grafana-up:
	@echo "Starting Grafana..."
	docker run -d -p 3000:3000 grafana/grafana

# Utility targets
check-deps:
	@echo "Checking for required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting." >&2; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting." >&2; exit 1; }
	@command -v migrate >/dev/null 2>&1 || { echo "golang-migrate is required. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" >&2; exit 1; }
	@echo "All required tools are installed"

install-tools:
	@echo "Installing development tools..."
	go install github.com/99designs/gqlgen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Development tools installed"

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download dependencies"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  test-integration- Run integration tests"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  generate        - Generate GraphQL code"
	@echo "  migrate-up      - Run database migrations"
	@echo "  migrate-down    - Rollback database migrations"
	@echo "  migrate-create  - Create new migration (use: make migrate-create name=migration_name)"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-push     - Push Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  dev-setup       - Setup development environment"
	@echo "  dev-run         - Start development server"
	@echo "  dev-ingest      - Start development ingester"
	@echo "  helm-install    - Install Helm chart"
	@echo "  helm-upgrade    - Upgrade Helm chart"
	@echo "  helm-uninstall  - Uninstall Helm chart"
	@echo "  check-deps      - Check for required tools"
	@echo "  install-tools   - Install development tools"
	@echo "  help            - Show this help message"
