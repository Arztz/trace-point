# QA Test scripts for trace-point

# Run all unit tests
test-unit:
	go test ./... -v

# Run unit tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run specific component tests
test-detector:
	go test -v ./internal/correlation/...

test-storage:
	go test -v ./internal/storage/...

test-config:
	go test -v ./internal/config/...

test-prometheus:
	go test -v ./internal/integration/prometheus/...

test-signoz:
	go test -v ./internal/integration/signoz/...

test-profiler:
	go test -v ./internal/integration/profiler/...

# Run all integration tests
test-integration: test-prometheus test-signoz test-profiler

# Run all tests (unit + integration)
test-all: test-unit test-integration

# Frontend tests
test-e2e:
	cd ui && npx playwright test

test-e2e-ui:
	cd ui && npx playwright test --ui

test-e2e-headed:
	cd ui && npx playwright test --headed

test-e2e-report:
	cd ui && npx playwright show-report

# Install Playwright browsers
test-install-browsers:
	cd ui && npx playwright install --with-deps

# Load testing
test-load:
	k6 run tests/load/load_test.js

test-load-spike:
	k6 run tests/load/spike_load_test.js

test-load-smoke:
	k6 run tests/load/load_test.js --out json=results.json

# Run load test with custom URL
TEST_URL=http://localhost:8080 test-load:
	k6 run tests/load/load_test.js -e BASE_URL=$(TEST_URL)

# Run Go linter
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run ./... --fix

# Format code
fmt:
	go fmt ./...
	cd ui && npm run format

# Build all
build:
	go build -o bin/trace-point ./cmd/server
	cd ui && npm run build

# Quick smoke test
smoke:
	go build -o bin/trace-point ./cmd/server
	./bin/trace-point &
	sleep 2
	curl -s http://localhost:8081/health | grep -q "OK" && echo "Server started"
	pkill trace-point || true

# Dev: Run backend
dev:
	go run ./cmd/server/main.go

# Dev: Run frontend
dev-ui:
	cd ui && npm run dev

# Dev: Run both
dev-all:
	@echo "Starting backend..."
	go run ./cmd/server &
	@echo "Starting frontend..."
	cd ui && npm run dev

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	cd ui && rm -rf playwright-report/ test-results/

# Complete test suite
test-complete: test-all test-e2e test-load

.PHONY: test-unit test-coverage test-detector test-storage test-config
.PHONY: test-prometheus test-signoz test-profiler test-integration test-all
.PHONY: test-e2e test-e2e-ui test-e2e-headed test-e2e-report
.PHONY: test-load test-load-spike test-load-smoke
.PHONY: lint lint-fix fmt build smoke dev dev-ui dev-all clean test-complete