APP_NAME := rfpak-explorer
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
BUILD_DIR := build

.PHONY: help build run clean deps build-all release package

help:
	@echo "RayFlow PAK Explorer (Go) Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  deps          - Download dependencies"
	@echo "  build         - Build for current platform"
	@echo "  run           - Run application"
	@echo "  build-all     - Build for all platforms"
	@echo "  release       - Build optimized release"
	@echo "  package       - Create native packages"
	@echo "  clean         - Remove build artifacts"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  make build-linux"
	@echo "  make build-windows"
	@echo "  make build-macos"
	@echo "  make build-macos-arm"

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

build: deps
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) .

run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

build-all: build-linux build-windows build-macos build-macos-arm
	@echo "All platforms built!"

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

build-macos:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

build-macos-arm:
	@echo "Building for macOS (arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

release: deps
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) .
	@if command -v upx > /dev/null; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma $(BUILD_DIR)/$(APP_NAME); \
	else \
		echo "UPX not found, skipping compression"; \
	fi

package:
	@echo "Creating native package..."
	@if ! command -v fyne > /dev/null; then \
		echo "Installing fyne CLI..."; \
		go install fyne.io/fyne/v2/cmd/fyne@latest; \
	fi
	fyne package -name "RayFlow PAK Explorer" -appVersion $(VERSION)

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME) $(APP_NAME).exe
	rm -rf *.app *.exe
	go clean

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

.DEFAULT_GOAL := help
