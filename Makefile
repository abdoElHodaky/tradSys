# TradSys Makefile - Comprehensive Build and Test Management

.PHONY: help build test test-unit test-integration test-performance test-compliance test-e2e
.PHONY: lint fmt vet clean deps docker-build docker-run
.PHONY: benchmark profile coverage security-scan
.PHONY: deploy-dev deploy-staging deploy-prod
.PHONY: proto migrate-up migrate-down

# Default target
help: ## Show this help message
	@echo "TradSys Build and Test Management"
	@echo "================================="
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the main application
	@echo "🔨 Building TradSys..."
	go build -v -o bin/tradsys ./cmd/tradsys

build-all: ## Build all services
	@echo "🔨 Building all services..."
	go build -v -o bin/tradsys ./cmd/tradsys
	@echo "✅ Build complete"

# Test targets
test: test-unit test-integration ## Run all tests
	@echo "✅ All tests completed"

test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	go test -v -race -coverprofile=coverage/unit.out ./tests/unit/...
	go test -v -race -coverprofile=coverage/components.out ./internal/... ./pkg/... ./services/...

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	go test -v -race -coverprofile=coverage/integration.out ./tests/integration/...

test-performance: ## Run performance tests
	@echo "⚡ Running performance tests..."
	go test -v -bench=. -benchmem -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof ./tests/performance/...

test-compliance: ## Run compliance validation tests
	@echo "🛡️ Running compliance tests..."
	go test -v -race ./tests/compliance/...

test-e2e: ## Run end-to-end tests
	@echo "🎯 Running end-to-end tests..."
	go test -v -timeout=30m ./tests/e2e/...

# Coverage and reporting
coverage: test-unit test-integration ## Generate coverage report
	@echo "📊 Generating coverage report..."
	mkdir -p coverage
	go tool cover -html=coverage/unit.out -o coverage/unit.html
	go tool cover -html=coverage/integration.out -o coverage/integration.html
	go tool cover -html=coverage/components.out -o coverage/components.html
	@echo "📊 Coverage reports generated in coverage/"

coverage-total: ## Calculate total coverage
	@echo "📊 Calculating total coverage..."
	go tool cover -func=coverage/components.out | tail -1

# Benchmarking and profiling
benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	mkdir -p benchmarks
	go test -bench=. -benchmem -count=5 ./tests/performance/... > benchmarks/results.txt
	@echo "⚡ Benchmark results saved to benchmarks/results.txt"

profile: ## Generate performance profiles
	@echo "📈 Generating performance profiles..."
	mkdir -p profiles
	go test -bench=. -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof ./tests/performance/...
	@echo "📈 Profiles generated in profiles/"

# Code quality
lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "🎨 Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

# Security
security-scan: ## Run security scan
	@echo "🔒 Running security scan..."
	gosec ./...
	nancy sleuth

# Dependencies
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy

deps-update: ## Update dependencies
	@echo "📦 Updating dependencies..."
	go get -u ./...
	go mod tidy

# Protocol Buffers
proto: ## Generate protobuf code
	@echo "🔄 Generating protobuf code..."
	./scripts/generate_proto.sh

# Database migrations
migrate-up: ## Run database migrations up
	@echo "⬆️ Running database migrations up..."
	go run cmd/migrate/main.go up

migrate-down: ## Run database migrations down
	@echo "⬇️ Running database migrations down..."
	go run cmd/migrate/main.go down

migrate-status: ## Check migration status
	@echo "📊 Checking migration status..."
	go run cmd/migrate/main.go status

# Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t tradsys:latest .

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker-compose up -d

docker-stop: ## Stop Docker containers
	@echo "🐳 Stopping Docker containers..."
	docker-compose down

# Deployment targets
deploy-dev: ## Deploy to development environment
	@echo "🚀 Deploying to development..."
	kubectl apply -f deployments/kubernetes/ -n tradsys-dev

deploy-staging: ## Deploy to staging environment
	@echo "🚀 Deploying to staging..."
	kubectl apply -f deployments/kubernetes/ -n tradsys-staging

deploy-prod: ## Deploy to production environment
	@echo "🚀 Deploying to production..."
	kubectl apply -f deployments/kubernetes/ -n tradsys-prod

# Cleanup
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf coverage/
	rm -rf profiles/
	rm -rf benchmarks/
	go clean -cache
	go clean -testcache

# Development helpers
dev-setup: deps proto ## Setup development environment
	@echo "🛠️ Setting up development environment..."
	mkdir -p {bin,coverage,profiles,benchmarks}
	@echo "✅ Development environment ready"

dev-test: fmt vet lint test-unit ## Run development tests
	@echo "✅ Development tests completed"

ci-test: deps proto fmt vet lint test coverage ## Run CI tests
	@echo "✅ CI tests completed"

# Load testing
load-test: ## Run load tests
	@echo "📈 Running load tests..."
	go test -v -timeout=10m ./tests/performance/load/...

stress-test: ## Run stress tests
	@echo "💪 Running stress tests..."
	go test -v -timeout=30m ./tests/performance/stress/...

# Monitoring and health checks
health-check: ## Check system health
	@echo "🏥 Checking system health..."
	curl -f http://localhost:8080/health || exit 1

metrics: ## Get system metrics
	@echo "📊 Getting system metrics..."
	curl -s http://localhost:8080/metrics

# Documentation
docs-serve: ## Serve documentation locally
	@echo "📚 Serving documentation..."
	@echo "Documentation available at: http://localhost:8000"
	python3 -m http.server 8000 -d docs/

docs-validate: ## Validate documentation
	@echo "📚 Validating documentation..."
	markdownlint docs/
	@echo "✅ Documentation validation complete"

# All-in-one targets
all: clean deps proto build test coverage ## Build and test everything
	@echo "🎉 Complete build and test cycle finished!"

ci: deps proto build ci-test security-scan ## Complete CI pipeline
	@echo "🎉 CI pipeline completed successfully!"

release: clean deps proto build test coverage security-scan ## Prepare release
	@echo "🎉 Release preparation completed!"
