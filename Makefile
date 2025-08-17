# Makefile for mitl development

.PHONY: all build build-all dist test clean clean-artifacts install fmt lint deps vendor docker-build docker-run dev watch test-coverage uninstall \
	test-unit test-integration test-race test-bench ci-local pre-commit preflight preflight-light

# Variables
BINARY_NAME=mitl
MAIN_PATH=cmd/mitl/main.go
BUILD_DIR=bin
INSTALL_PATH=/usr/local/bin
DIST_DIR=dist

# Build flags
LDFLAGS=-ldflags "-X mitl/pkg/version.Version=$$(git describe --tags --always 2>/dev/null || echo dev)"

all: test build

build:
	@echo "Building mitl..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all:
	@echo "Building mitl for darwin/amd64 and darwin/arm64..."
	@mkdir -p $(BUILD_DIR)
	@env GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/mitl-darwin-amd64 $(MAIN_PATH)
	@env GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/mitl-darwin-arm64 $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/mitl-darwin-amd64, $(BUILD_DIR)/mitl-darwin-arm64"

dist: build-all
	@echo "Packaging release tarballs..."
	@mkdir -p $(DIST_DIR)
	@mkdir -p $(BUILD_DIR)/pkg-darwin-amd64 $(BUILD_DIR)/pkg-darwin-arm64
	@cp $(BUILD_DIR)/mitl-darwin-amd64 $(BUILD_DIR)/pkg-darwin-amd64/mitl
	@cp $(BUILD_DIR)/mitl-darwin-arm64 $(BUILD_DIR)/pkg-darwin-arm64/mitl
	@tar -C $(BUILD_DIR)/pkg-darwin-amd64 -czf $(DIST_DIR)/mitl-darwin-amd64.tar.gz mitl
	@tar -C $(BUILD_DIR)/pkg-darwin-arm64 -czf $(DIST_DIR)/mitl-darwin-arm64.tar.gz mitl
	@echo "Artifacts: $(DIST_DIR)/mitl-darwin-amd64.tar.gz, $(DIST_DIR)/mitl-darwin-arm64.tar.gz"

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

# Additional testing targets
test-unit:
	@echo "Running unit tests..."
	@go test -v -short ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -run Integration ./...

test-race:
	@echo "Running tests with race detection..."
	@go test -race ./...

test-bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/digest

test-coverage: test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@echo "Formatting complete"

install: build
	@echo "Installing mitl to $(INSTALL_PATH)..."
	@install -m 0755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete"

uninstall:
	@echo "Removing mitl from $(INSTALL_PATH)..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstall complete"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

clean-artifacts: clean
	@echo "Removing additional local artifacts..."
	@rm -f container.out || true
	@echo "Artifact cleanup complete"

dev: fmt test build

watch:
	@echo "Watching for changes..."
	@fswatch -o . | xargs -n1 -I{} make build

deps:
	@go mod download
	@go mod tidy

vendor:
	@go mod vendor

# CI simulation locally
ci-local:
	@echo "Running CI checks locally..."
	@$(MAKE) fmt
	@$(MAKE) test
	@$(MAKE) build
	@echo "✅ All CI checks passed!"

# Pre-commit quick check
pre-commit:
	@go fmt ./...
	@go vet ./...
	@go test -short ./...
	@echo "✅ Ready to commit!"

preflight: build
	@echo "Running full preflight checks..."
	@PATH="$(PWD)/$(BUILD_DIR):$$PATH" bash scripts/preflight.sh

preflight-light: build
	@echo "Running light preflight checks..."
	@PATH="$(PWD)/$(BUILD_DIR):$$PATH" bash scripts/preflight.sh --light --skip-docker

docker-build:
	@docker build -t mitl:latest .

docker-run:
	@docker run --rm -it mitl:latest

.DEFAULT_GOAL := build
