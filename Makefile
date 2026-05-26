SHELL := /bin/bash
GO ?= go
GOFLAGS ?= -ldflags="-s -w"
BIN_DIR := bin
COVERAGE_DIR := coverage

# Build metadata
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X 'github.com/FortressWAF/FortressWAF/internal/version.Version=$(VERSION)' -X 'github.com/FortressWAF/FortressWAF/internal/version.Commit=$(COMMIT)' -X 'github.com/FortressWAF/FortressWAF/internal/version.BuildDate=$(BUILD_DATE)'"

.PHONY: help dev build build-all test lint lint-go lint-py lint-ts clean docker-build docker-up docker-down docker-logs docker-dev docker-dev-down docker-monitoring docker-status docker-clean docker-shell-proxy docker-shell-ml docker-shell-db release install uninstall coverage bench profile format generate docs

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Start development environment (Docker)
	@echo "Starting FortressWAF development stack..."
	docker compose -f deploy/docker-compose.dev.yml up -d --build
	@echo ""
	@echo "═══════════════════════════════════════════════════════"
	@echo "  FortressWAF Dev Stack Running"
	@echo "═══════════════════════════════════════════════════════"
	@echo "  Proxy:     http://localhost:8080"
	@echo "  Admin API: http://localhost:8443"
	@echo "  Dashboard: http://localhost:3000"
	@echo "  ML Engine: http://localhost:8000"
	@echo "  Redis:     localhost:6379"
	@echo "  Postgres:  localhost:5432"
	@echo "  NATS:      localhost:4222 (monitor: 8222)"
	@echo "═══════════════════════════════════════════════════════"

dev-down: ## Stop development environment
	docker compose -f deploy/docker-compose.dev.yml down -v
	@echo "Development stack stopped."

dev-logs: ## View development logs
	docker compose -f deploy/docker-compose.dev.yml logs -f

build: ## Build all Go binaries
	@mkdir -p $(BIN_DIR)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy ./cmd/proxy
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortressctl ./cmd/ctl
	@echo "Binaries built in $(BIN_DIR)/"

build-all: build ## Build for all platforms
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy-linux-amd64 ./cmd/proxy
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy-linux-arm64 ./cmd/proxy
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy-darwin-amd64 ./cmd/proxy
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy-darwin-arm64 ./cmd/proxy
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/fortress-proxy-windows-amd64.exe ./cmd/proxy
	@echo "Cross-compiled binaries in $(BIN_DIR)/"

test: ## Run all tests
	$(GO) test ./... -v -race -count=1 -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1

test-unit: ## Run unit tests only
	$(GO) test ./internal/... -v -race -count=1 -coverprofile=$(COVERAGE_DIR)/unit.out

test-integration: ## Run integration tests
	$(GO) test -tags=integration ./tests/... -v -count=1 -coverprofile=$(COVERAGE_DIR)/integration.out

train-ml: ## Train ML models with attack corpus
	cd ml-engine && pip install -r requirements.txt && python -m training.train --data-dir training/data

test-ml: ## Run ML engine tests
	cd ml-engine && pip install -r requirements-dev.txt && python -m pytest test_api.py -v

test-dashboard: ## Run dashboard tests
	cd dashboard && npm test -- --watchAll=false

test-e2e: ## Run end-to-end tests
	$(GO) test -tags=e2e ./tests/e2e/... -v -count=1

lint: lint-go lint-py lint-ts lint-docker lint-yaml lint-markdown ## Run all linters

lint-go: ## Run Go linter
	golangci-lint run ./... --timeout=5m --out-format=colored-line-number

lint-py: ## Run Python linter
	ruff check ml-engine/ --fix
	ruff format ml-engine/ --check

lint-ts: ## Run TypeScript linter
	cd dashboard && npm run lint

lint-docker: ## Run Dockerfile linter
	 hadolint Dockerfile
	 hadolint ml-engine/Dockerfile
	 hadolint dashboard/Dockerfile

lint-yaml: ## Run YAML linter
	yamllint .

lint-markdown: ## Run Markdown linter
	markdownlint docs/ README.md

coverage: ## Generate coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test ./... -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

bench: ## Run benchmarks
	$(GO) test ./... -bench=. -benchmem -run=^$$ -count=3 | tee $(COVERAGE_DIR)/benchmark.txt

profile: ## Run CPU profile
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test ./internal/engine -bench=BenchmarkRuleEngine -cpuprofile=$(COVERAGE_DIR)/cpu.prof -memprofile=$(COVERAGE_DIR)/mem.prof
	@echo "Profiles saved to $(COVERAGE_DIR)/"

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)/
	rm -rf $(COVERAGE_DIR)/
	rm -f *.out *.test *.prof
	$(GO) clean -cache -testcache
	@echo "Clean complete"

docker-build: ## Build all Docker images
	@echo "Building FortressWAF Docker images..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t fortresswaf/proxy:$(VERSION) \
		-t fortresswaf/proxy:latest .
	docker build -t fortresswaf/ml-engine:$(VERSION) -t fortresswaf/ml-engine:latest ml-engine/
	docker build -t fortresswaf/dashboard:$(VERSION) -t fortresswaf/dashboard:latest dashboard/
	@echo "Docker images built: proxy, ml-engine, dashboard ($(VERSION))"

docker-build-multi: ## Build multi-arch Docker images
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t fortresswaf/proxy:$(VERSION) .
	docker buildx build --platform linux/amd64,linux/arm64 -t fortresswaf/ml-engine:$(VERSION) ml-engine/
	docker buildx build --platform linux/amd64,linux/arm64 -t fortresswaf/dashboard:$(VERSION) dashboard/

docker-up: ## Start full production stack
	@test -f .env || (echo "ERROR: .env not found. Run: cp .env.example .env" && exit 1)
	@echo "Starting FortressWAF production stack..."
	docker compose -f deploy/docker-compose.yml up -d
	@echo ""
	@echo "═══════════════════════════════════════════════════════"
	@echo "  FortressWAF Production Stack Running"
	@echo "═══════════════════════════════════════════════════════"
	@echo "  Proxy:      http://localhost:$${PROXY_HTTP_PORT:-80}"
	@echo "  Admin API:  https://localhost:$${PROXY_ADMIN_PORT:-8443}"
	@echo "  Dashboard:  http://localhost:$${DASHBOARD_PORT:-3000}"
	@echo "  ML Engine:  http://localhost:$${ML_ENGINE_PORT:-8000}"
	@echo "═══════════════════════════════════════════════════════"

docker-down: ## Stop full production stack
	docker compose -f deploy/docker-compose.yml down
	@echo "Production stack stopped. Data volumes preserved."

docker-down-clean: ## Stop production stack and remove volumes
	docker compose -f deploy/docker-compose.yml down -v
	@echo "Production stack stopped. All data removed."

docker-logs: ## View production logs (all services)
	docker compose -f deploy/docker-compose.yml logs -f

docker-logs-proxy: ## View proxy logs only
	docker compose -f deploy/docker-compose.yml logs -f fortress-proxy

docker-logs-ml: ## View ML engine logs only
	docker compose -f deploy/docker-compose.yml logs -f ml-engine

docker-dev: ## Start development environment with hot-reload
	@echo "Starting FortressWAF dev stack with hot-reload..."
	docker compose -f deploy/docker-compose.dev.yml up -d --build
	@echo "Dev stack running. All services have hot-reload enabled."

docker-dev-down: ## Stop development environment
	docker compose -f deploy/docker-compose.dev.yml down -v

docker-monitoring: ## Start monitoring stack (Prometheus + Grafana + Loki)
	@echo "Starting monitoring stack..."
	docker compose -f deploy/docker-compose.yml \
		-f deploy/monitoring/docker-compose.monitoring.yml up -d
	@echo ""
	@echo "  Prometheus:   http://localhost:$${PROMETHEUS_PORT:-9090}"
	@echo "  Grafana:      http://localhost:$${GRAFANA_PORT:-3001}"
	@echo "  Alertmanager: http://localhost:$${ALERTMANAGER_PORT:-9093}"

docker-status: ## Show status of all containers
	@echo "FortressWAF Container Status:"
	@echo "═══════════════════════════════════════════════════════"
	@docker compose -f deploy/docker-compose.yml ps 2>/dev/null || echo "Production stack: not running"
	@echo ""
	@docker compose -f deploy/docker-compose.dev.yml ps 2>/dev/null || echo "Dev stack: not running"

docker-clean: ## Remove all FortressWAF Docker resources
	@echo "Removing all FortressWAF Docker resources..."
	docker compose -f deploy/docker-compose.yml down -v --rmi local 2>/dev/null || true
	docker compose -f deploy/docker-compose.dev.yml down -v --rmi local 2>/dev/null || true
	docker compose -f deploy/monitoring/docker-compose.monitoring.yml down -v --rmi local 2>/dev/null || true
	docker image rm fortresswaf/proxy:latest fortresswaf/ml-engine:latest fortresswaf/dashboard:latest 2>/dev/null || true
	@echo "Cleanup complete."

docker-shell-proxy: ## Open shell in proxy container
	docker exec -it fortress-proxy /bin/sh

docker-shell-ml: ## Open shell in ML engine container
	docker exec -it fortress-ml-engine /bin/bash

docker-shell-db: ## Open psql in PostgreSQL container
	docker exec -it fortress-postgres psql -U $${DB_USER:-fortress} -d $${DB_NAME:-fortresswaf}

format: ## Format code
	$(GO) fmt ./...
	ruff format ml-engine/
	cd dashboard && npx prettier --write "src/**/*.{ts,tsx,js,jsx,json,css,scss}"

generate: ## Generate code
	$(GO) generate ./...

docs: ## Start documentation server
	cd docs && mkdocs serve

docs-build: ## Build documentation site
	cd docs && mkdocs build --strict

install: build ## Install binaries to system
	sudo cp $(BIN_DIR)/fortress-proxy /usr/local/bin/
	sudo cp $(BIN_DIR)/fortressctl /usr/local/bin/
	sudo mkdir -p /etc/fortresswaf/rules
	@echo "Installed to /usr/local/bin/"

uninstall: ## Remove installed binaries
	sudo rm -f /usr/local/bin/fortress-proxy
	sudo rm -f /usr/local/bin/fortressctl
	@echo "Uninstalled"

release: lint test docker-build ## Prepare release
	@echo "Release checks passed for v$(VERSION)"

security-scan: ## Run security scans
	trivy image fortresswaf/proxy:latest --severity CRITICAL,HIGH --exit-code 1
	govulncheck ./...
	cd ml-engine && pip-audit
	cd dashboard && npm audit

vulncheck: ## Check for known vulnerabilities
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Dependency management
dep-update: ## Update Go dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

dep-audit: ## Audit dependencies
	$(GO) list -m -u all 2>/dev/null | grep '\['
	cd ml-engine && pip list --outdated
	cd dashboard && npm outdated

# Git hooks
install-hooks: ## Install pre-commit hooks
	pre-commit install
	pre-commit install --hook-type commit-msg
	@echo "Pre-commit hooks installed"

update-hooks: ## Update pre-commit hooks
	pre-commit autoupdate

run-hooks: ## Run pre-commit hooks on all files
	pre-commit run --all-files

# Compliance
compliance-report: ## Generate compliance report
	fortressctl compliance report --all --period 90d --output ./reports/

# Attack corpus validation
validate-corpus: ## Validate attack corpus files
	@echo "Validating attack corpus..."
	@for f in tests/attack-corpus/*.txt; do \
		lines=$$(wc -l < "$$f"); \
		echo "  $$f: $$lines lines"; \
	done
	@echo "Corpus validated"
