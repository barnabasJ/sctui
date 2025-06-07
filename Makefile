# SoundCloud TUI Makefile

.PHONY: build test clean run help

# Build the main application
build:
	@echo "Building sctui..."
	@go build -o bin/sctui ./cmd/sctui

# Build test application
build-test:
	@echo "Building test application..."
	@go build -o bin/test ./cmd/test

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Run the application with search example
run:
	@make build
	@echo "Running example search..."
	@./bin/sctui -search "lofi"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Show available commands
help:
	@echo "Available commands:"
	@echo "  build       - Build the main sctui application"
	@echo "  build-test  - Build the test application" 
	@echo "  test        - Run all tests"
	@echo "  clean       - Remove build artifacts"
	@echo "  run         - Build and run example search"
	@echo "  deps        - Install and tidy dependencies"
	@echo "  help        - Show this help message"

# Default target
all: clean build