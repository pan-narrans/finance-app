.PHONY: build test run clean build-webapp

# Variables
WEBAPP_DIR := internal/adapters/primary/telegram/webapp
BINARY_NAME := finance-app
TMP_DIR := tmp

# Default target
all: build

# Build the React frontend
build-webapp:
	@echo "Building WebApp frontend..."
	@cd $(WEBAPP_DIR) && npm install --silent && npm run build --silent

# Build the complete Go binary (requires frontend assets for go:embed)
build: build-webapp
	@echo "Building Go backend..."
	@go build -o $(TMP_DIR)/$(BINARY_NAME) ./cmd/finance-app/main.go
	@echo "Build complete: $(TMP_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run the application locally
run: build
	@echo "Starting application..."
	@./$(TMP_DIR)/$(BINARY_NAME)

# Local development with Cloudflare Tunnel and Air
dev:
	@./dev.sh

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(TMP_DIR)
	@rm -rf $(WEBAPP_DIR)/dist
	@rm -rf $(WEBAPP_DIR)/node_modules
	@echo "Clean complete."
