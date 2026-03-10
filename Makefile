.PHONY: build dev test clean migrate

# Build the complete application (frontend + Go binary)
build:
	@echo "Building frontend..."
	cd frontend && npm ci && npm run build
	@echo "Building Go binary..."
	CGO_ENABLED=0 go build -o zfaktury ./cmd/zfaktury
	@echo "Build complete: ./zfaktury"

# Run in development mode with hot reloading
dev:
	@bash scripts/dev.sh

# Run all tests (backend + frontend)
test:
	CGO_ENABLED=0 go test ./... -v
	cd frontend && npm run test

# Clean build artifacts
clean:
	rm -f zfaktury
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf tmp/

# Run database migrations only
migrate:
	go run ./cmd/zfaktury serve --port 0 2>&1 | head -5
