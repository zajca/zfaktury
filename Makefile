.PHONY: build build-server dev test clean migrate lint lint-go lint-frontend format coverage coverage-go coverage-frontend coverage-critical install-hooks release release-retry

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
  -X github.com/zajca/zfaktury/internal/version.Version=$(VERSION) \
  -X github.com/zajca/zfaktury/internal/version.Commit=$(COMMIT) \
  -X github.com/zajca/zfaktury/internal/version.Date=$(DATE)

# Build desktop application (requires CGO + webkit2gtk)
build:
	@echo "Building frontend..."
	cd frontend && npm ci && npm run build
	rm -rf web/frontend/build && cp -r frontend/build web/frontend/build
	@echo "Building desktop binary (CGO required)..."
	go build -ldflags "$(LDFLAGS)" -o zfaktury ./cmd/zfaktury
	@echo "Build complete: ./zfaktury"

# Build server-only binary (no CGO, no native GUI deps)
build-server:
	@echo "Building frontend..."
	cd frontend && npm ci && npm run build
	rm -rf web/frontend/build && cp -r frontend/build web/frontend/build
	@echo "Building server binary (no CGO)..."
	CGO_ENABLED=0 go build -tags server -ldflags "$(LDFLAGS)" -o zfaktury-server ./cmd/zfaktury
	@echo "Build complete: ./zfaktury-server"

# Run in development mode with hot reloading
dev:
	@bash scripts/dev.sh

# Run all tests (backend + frontend)
test:
	CGO_ENABLED=0 go test -tags server ./... -v
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

# Coverage for critical financial/XML packages
coverage-critical:
	@echo "Checking critical package coverage..."
	@CGO_ENABLED=0 go test ./internal/calc/ -coverprofile=/tmp/calc.cov -count=1 > /dev/null
	@printf "calc:          " && go tool cover -func=/tmp/calc.cov | tail -1 | awk '{print $$NF}'
	@CGO_ENABLED=0 go test ./internal/vatxml/ -coverprofile=/tmp/vatxml.cov -count=1 > /dev/null
	@printf "vatxml:        " && go tool cover -func=/tmp/vatxml.cov | tail -1 | awk '{print $$NF}'
	@CGO_ENABLED=0 go test ./internal/annualtaxxml/ -coverprofile=/tmp/annualtaxxml.cov -count=1 > /dev/null
	@printf "annualtaxxml:  " && go tool cover -func=/tmp/annualtaxxml.cov | tail -1 | awk '{print $$NF}'
	@CGO_ENABLED=0 go test ./internal/isdoc/ -coverprofile=/tmp/isdoc.cov -count=1 > /dev/null
	@printf "isdoc:         " && go tool cover -func=/tmp/isdoc.cov | tail -1 | awk '{print $$NF}'

# Clean build artifacts
clean:
	rm -f zfaktury
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf tmp/
	rm -f coverage.out coverage.html
	rm -rf frontend/coverage/

# Install git hooks (points git to scripts/ directory)
install-hooks:
	git config core.hooksPath scripts
	@echo "Git hooks installed (core.hooksPath = scripts)."

# Run database migrations only
migrate:
	go run ./cmd/zfaktury serve --port 0 2>&1 | head -5

# Create a release tag (usage: make release V=v1.0.0)
release:
	@bash scripts/release.sh $(V)

# Retry a failed release (usage: make release-retry V=v1.0.0)
release-retry:
	@bash scripts/release.sh --retry $(V)
