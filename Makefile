.PHONY: build run test clean docker-up docker-down migrate help lint-install security coverage audit

# Database configuration (override with environment variables)
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= monitoring_db
DB_URL = postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Build the application
build:
	@echo "Building monitoring engine..."
	@go build -o bin/monitoring-tool cmd/monitoring-tool/*.go
	@echo "Build complete: bin/monitoring-tool"

# Run the application
run:
	@echo "Starting monitoring engine..."
	@POSTGRES_PASSWORD=$(DB_PASSWORD) go run cmd/monitoring-tool/*.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@docker-compose ps

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "Services stopped"

# Stop Docker services and remove volumes
docker-clean:
	@echo "Cleaning Docker services and volumes..."
	@docker-compose down -v
	@echo "Docker cleanup complete"

# Run database migrations
migrate-up:
	@echo "Running database migrations..."
	@migrate -path migrations -database "$(DB_URL)" up
	@echo "Migrations complete"

# Rollback migrations
migrate-down:
	@echo "Rolling back migrations..."
	@migrate -path migrations -database "$(DB_URL)" down
	@echo "Rollback complete"

# Create new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# Force migration version
migrate-force:
	@read -p "Enter version to force: " version; \
	migrate -path migrations -database "$(DB_URL)" force $$version

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Go dependencies installed"

# Install development tools
dev-tools:
	@echo "Installing development tools..."
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin)
	@command -v gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@command -v ginkgo > /dev/null || (echo "Installing ginkgo..." && go install github.com/onsi/ginkgo/v2/ginkgo@latest)
	@echo "Development tools installed"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# Run linter
lint:
	@echo "Running linter..."
	@command -v golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make dev-tools' to install." && exit 1)
	@golangci-lint run ./...

# Run security scanner
security:
	@echo "Running security scanner..."
	@command -v gosec > /dev/null || (echo "gosec not found. Run 'make dev-tools' to install." && exit 1)
	@gosec -fmt=json -out=security-report.json ./...
	@gosec ./...
	@echo "Security report generated: security-report.json"

# Generate detailed coverage report
coverage:
	@echo "Generating detailed coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "Coverage report: coverage.html"

# Run comprehensive audit
audit: lint security coverage
	@echo ""
	@echo "=== Audit Complete ==="
	@echo "Linting: ✓"
	@echo "Security: ✓ (see security-report.json)"
	@echo "Coverage: ✓ (see coverage.html)"

# Full setup (Docker + Migrate + Build)
setup: docker-up migrate-up build
	@echo "Setup complete! Run 'make run' to start the application."

# Development workflow
dev: docker-up
	@echo "Starting in development mode..."
	@POSTGRES_PASSWORD=$(DB_PASSWORD) go run cmd/monitoring-tool/*.go

# Show help
help:
	@echo "Monitoring Tool - Makefile Commands"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build the application binary"
	@echo "  make run            - Run the application"
	@echo "  make dev            - Start in development mode with Docker"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make coverage       - Generate detailed coverage report"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make security       - Run gosec security scanner"
	@echo "  make audit          - Run full audit (lint + security + coverage)"
	@echo "  make fmt            - Format code with go fmt"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback migrations"
	@echo "  make migrate-create - Create new migration"
	@echo "  make migrate-force  - Force migration to specific version"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up      - Start Docker services (PostgreSQL)"
	@echo "  make docker-down    - Stop Docker services"
	@echo "  make docker-clean   - Stop services and remove volumes"
	@echo ""
	@echo "Setup & Dependencies:"
	@echo "  make setup          - Full setup (Docker + Migrate + Build)"
	@echo "  make deps           - Install Go dependencies"
	@echo "  make dev-tools      - Install development tools (golangci-lint, gosec, ginkgo)"
	@echo ""
	@echo "Configuration:"
	@echo "  Override database connection with environment variables:"
	@echo "    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME"
	@echo "  Example: DB_PASSWORD=secret make migrate-up"
