# Variables
BINARY_NAME=guardz
BINARY_UNIX=$(BINARY_NAME)_unix
DOCKER_IMAGE=guardz
DOCKER_TAG=latest
DOCKER_USERNAME?=your-dockerhub-username
DOCKER_REPO=$(DOCKER_USERNAME)/$(DOCKER_IMAGE)
PORT=8080

# Build info variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT  ?= $(shell git rev-parse --short HEAD)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_DIR=bin

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: all build clean run test deps docker-build docker-run docker-stop docker-push docker-build-push docker-clean help

## Default: run all
all: clean deps test build

## Build the application
build:
	@echo "Building..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'" -o $(BINARY_DIR)/$(BINARY_NAME) -v ./cmd/main.go

## Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BINARY_DIR)
	@go clean -testcache

## Run the application
run:
	@echo "Running..."
	$(GOBUILD) -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'" -o $(BINARY_DIR)/$(BINARY_NAME) -v ./cmd/main.go
	./$(BINARY_DIR)/$(BINARY_NAME)

## Test the application
test:
	@echo "Testing..."
	$(GOTEST) -v ./...

## Test with coverage
test-coverage:
	@echo "Testing with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Get dependencies
deps:
	@echo "Getting dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Install dependencies
install-deps:
	@echo "No external dependencies to install"

## Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## Run linter
lint:
	@echo "Linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi


## Docker build
docker-build:
	@echo "Building Docker image..."
	docker build --build-arg PORT=$(PORT) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REPO):$(DOCKER_TAG)

## Docker run
docker-run:docker-build
	@echo "Running Docker container..."
	docker run -p $(PORT):$(PORT) --name $(BINARY_NAME) \
		-e IP_DB_CONFIG='{"dbtype": "csv", "extra_details": {"file_path": "/app/TestFiles/ip_data.csv"}}' \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## Docker stop
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

## Docker push
docker-push:
	@echo "Pushing Docker image to Docker Hub..."
	@if [ "$(DOCKER_USERNAME)" = "your-dockerhub-username" ]; then \
		echo "Error: Please set DOCKER_USERNAME environment variable or update Makefile"; \
		echo "Usage: make docker-push DOCKER_USERNAME=your-username"; \
		exit 1; \
	fi
	docker push $(DOCKER_REPO):$(DOCKER_TAG)

## Docker build and push
docker-build-push: docker-build docker-push

## Docker clean
docker-clean:
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true
	docker rmi $(DOCKER_REPO):$(DOCKER_TAG) || true

## Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not found. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

## Show help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Run the application"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  deps          - Get dependencies"
	@echo "  install-deps  - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-stop   - Stop Docker container"
	@echo "  docker-push   - Push Docker image to Docker Hub"
	@echo "  docker-build-push - Build and push Docker image"
	@echo "  docker-clean  - Clean Docker images"
	@echo "  security      - Run security scan"
	@echo "  help          - Show this help message"