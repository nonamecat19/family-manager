.PHONY: build run migrate-up migrate-down sqlc generate test clean

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Check if migrate is installed
check-migrate:
	@which migrate > /dev/null || (echo "❌ Error: 'migrate' command not found." && \
		echo "📦 Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && \
		echo "   Then add $$(go env GOPATH)/bin to your PATH" && \
		echo "   Or run from root: make install-tools" && \
		exit 1)

# Default database URL (matches docker-compose postgres setup)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/finance_manager?sslmode=disable

# Run database migrations up
migrate-up: check-migrate
	@if [ -z "$$DATABASE_URL" ]; then \
		echo "⚠️  DATABASE_URL not set, using default: $(DATABASE_URL)"; \
	fi
	migrate -path migrations -database "$${DATABASE_URL:-$(DATABASE_URL)}" up

# Run database migrations down
migrate-down: check-migrate
	@if [ -z "$$DATABASE_URL" ]; then \
		echo "⚠️  DATABASE_URL not set, using default: $(DATABASE_URL)"; \
	fi
	migrate -path migrations -database "$${DATABASE_URL:-$(DATABASE_URL)}" down

# Generate SQLC code
sqlc:
	sqlc generate

# Generate GraphQL code
generate:
	go run github.com/99designs/gqlgen generate

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

