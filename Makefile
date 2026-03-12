.PHONY: build dev test clean migrate lint lint-go lint-frontend format coverage coverage-go coverage-frontend install-hooks

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
  -X github.com/zajca/zfaktury/internal/version.Version=$(VERSION) \
  -X github.com/zajca/zfaktury/internal/version.Commit=$(COMMIT) \
  -X github.com/zajca/zfaktury/internal/version.Date=$(DATE)

# Build the complete application (frontend + Go binary)
build:
	@echo "Building frontend..."
	cd frontend && npm ci && npm run build
	@echo "Building Go binary..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o zfaktury ./cmd/zfaktury
	@echo "Build complete: ./zfaktury"

# Run in development mode with hot reloading
dev:
	@bash scripts/dev.sh

# Run all tests (backend + frontend)
test:
	CGO_ENABLED=0 go test ./... -v
	cd frontend && npm run test

# Lint all code
lint: lint-go lint-frontend

lint-go:
	golangci-lint run ./...

lint-frontend:
	cd frontend && npm run lint
	cd frontend && npm run check
	cd frontend && npm run format:check

# Format frontend code
format:
	cd frontend && npm run lint:fix
	cd frontend && npm run format

# Code coverage
coverage: coverage-go coverage-frontend

coverage-go:
	CGO_ENABLED=0 go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

coverage-frontend:
	cd frontend && npm run test:coverage

# Clean build artifacts
clean:
	rm -f zfaktury
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf tmp/
	rm -f coverage.out coverage.html
	rm -rf frontend/coverage/

# Install git hooks
install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed."

# Run database migrations only
migrate:
	go run ./cmd/zfaktury serve --port 0 2>&1 | head -5
