# Media Converter Build System

# Build variables
BINARY_NAME=media-converter
MAIN_PATH=./main.go
BUILD_DIR=./build

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test deps run help install cross-compile

## Build commands

all: clean deps test build ## Run all build steps

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-local: ## Build for local development
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

install: ## Install the binary to GOPATH/bin
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)

## Cross-compilation

cross-compile: ## Build for multiple platforms (detailed names)
	@echo "Cross-compiling..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Cross-compilation complete! Binaries in $(BUILD_DIR)/"

release-binaries: ## Build release binaries with architecture-specific names for GitHub
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	# macOS builds
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-intel $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-apple-silicon $(MAIN_PATH)
	# Linux builds
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	# Windows build
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo ""
	@echo "✅ Release binaries ready in $(BUILD_DIR)/:"
	@echo "   • $(BINARY_NAME)-macos-intel (macOS Intel x64)"
	@echo "   • $(BINARY_NAME)-macos-apple-silicon (macOS M1/M2/M3/M4)"
	@echo "   • $(BINARY_NAME)-linux-amd64 (Linux x64)"
	@echo "   • $(BINARY_NAME)-linux-arm64 (Linux ARM64)"
	@echo "   • $(BINARY_NAME)-windows-amd64.exe (Windows x64)"
	@echo ""
	@echo "Ready to upload to GitHub Releases!"

## Development commands

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

run: build-local ## Build and run with example parameters
	@echo "Running $(BINARY_NAME) in dry-run mode..."
	./$(BINARY_NAME) --dry-run --help

## Release commands

release-prep: ## Prepare for release (clean, test, release-binaries)
	@echo "Preparing release..."
	$(MAKE) clean
	$(MAKE) deps
	$(MAKE) test
	$(MAKE) release-binaries
	@echo "Release preparation complete!"

## Docker commands (future)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

## Documentation

docs: ## Generate documentation
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.md

## Linting and formatting

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

lint: fmt vet ## Run all linting tools

## Development workflow

dev: clean deps build-local ## Quick development build

check: fmt vet test ## Run all checks

## Help

help: ## Show this help message
	@echo "Media Converter Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Default target
.DEFAULT_GOAL := help