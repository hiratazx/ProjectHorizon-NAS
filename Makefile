.PHONY: all build clean run dev install uninstall

# Variables
APP_NAME := horizon
VERSION := 0.1.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DIR := build
SYSROOT_DIR := $(BUILD_DIR)/sysroot
INSTALL_PREFIX := /usr/local
SERVICE_NAME := projecthorizon

# Go build settings
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
all: build

# Build the application
build: clean
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(SYSROOT_DIR)$(INSTALL_PREFIX)/bin
	@mkdir -p $(SYSROOT_DIR)/etc/projecthorizon
	@mkdir -p $(SYSROOT_DIR)/var/lib/projecthorizon/www
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/horizon
	@cp $(BUILD_DIR)/$(APP_NAME) $(SYSROOT_DIR)$(INSTALL_PREFIX)/bin/
	@cp -r public/* $(SYSROOT_DIR)/var/lib/projecthorizon/www/ 2>/dev/null || true
	@cp -r config $(SYSROOT_DIR)/etc/projecthorizon/
	@cp scripts/projecthorizon.service $(SYSROOT_DIR)/etc/systemd/system/ 2>/dev/null || mkdir -p $(SYSROOT_DIR)/etc/systemd/system && cp scripts/projecthorizon.service $(SYSROOT_DIR)/etc/systemd/system/
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/horizon
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/horizon
	GOOS=linux GOARCH=arm go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm ./cmd/horizon

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Run in development mode
dev:
	@go run ./cmd/horizon

# Run the built binary
run: build
	@./$(BUILD_DIR)/$(APP_NAME)

# Install to system
install: build
	@echo "Installing $(APP_NAME)..."
	@sudo cp -r $(SYSROOT_DIR)/* /
	@sudo systemctl daemon-reload
	@sudo systemctl enable $(SERVICE_NAME)
	@sudo systemctl start $(SERVICE_NAME)
	@echo "$(APP_NAME) installed and started!"

# Uninstall from system
uninstall:
	@echo "Uninstalling $(APP_NAME)..."
	@sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true
	@sudo systemctl disable $(SERVICE_NAME) 2>/dev/null || true
	@sudo rm -f $(INSTALL_PREFIX)/bin/$(APP_NAME)
	@sudo rm -rf /etc/projecthorizon
	@sudo rm -rf /var/lib/projecthorizon
	@sudo rm -f /etc/systemd/system/$(SERVICE_NAME).service
	@sudo systemctl daemon-reload
	@echo "$(APP_NAME) uninstalled!"

# Generate sysroot tarball for distribution
dist: build
	@echo "Creating distribution tarball..."
	@mkdir -p $(BUILD_DIR)/dist
	@tar -czvf $(BUILD_DIR)/dist/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C $(BUILD_DIR) sysroot
	@echo "Created: $(BUILD_DIR)/dist/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"

# Tidy dependencies
tidy:
	@go mod tidy

# Run tests
test:
	@go test -v ./...

# Format code
fmt:
	@go fmt ./...

# Lint code
lint:
	@golangci-lint run

# Show help
help:
	@echo "ProjectHorizon Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the application"
	@echo "  build-all   Build for all platforms (linux amd64/arm64/arm)"
	@echo "  clean       Clean build artifacts"
	@echo "  dev         Run in development mode"
	@echo "  run         Build and run"
	@echo "  install     Install to system with systemd service"
	@echo "  uninstall   Remove from system"
	@echo "  dist        Create distribution tarball"
	@echo "  tidy        Tidy go modules"
	@echo "  test        Run tests"
	@echo "  fmt         Format code"
	@echo "  help        Show this help"
