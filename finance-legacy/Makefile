.PHONY: help postgres-up postgres-down postgres-status start-local backend-install backend-migrate backend-run frontend-install frontend-dev install-tools

# Default target
help:
	@echo "Finance Manager - Local Development"
	@echo ""
	@echo "Available targets:"
	@echo "  make postgres-up        - Start PostgreSQL (docker-compose)"
	@echo "  make postgres-down      - Stop PostgreSQL"
	@echo "  make postgres-status    - Check PostgreSQL status"
	@echo "  make start-local        - Start PostgreSQL and show instructions"
	@echo ""
	@echo "Backend:"
	@echo "  make install-tools      - Install required tools (migrate, sqlc, gqlgen)"
	@echo "  make backend-install    - Install backend dependencies"
	@echo "  make backend-migrate    - Run database migrations"
	@echo "  make backend-run        - Run backend server"
	@echo ""
	@echo "Frontend:"
	@echo "  make frontend-install   - Install frontend dependencies"
	@echo "  make frontend-dev       - Run frontend dev server"
	@echo ""

# PostgreSQL targets
postgres-up:
	@echo "📦 Starting PostgreSQL..."
	@docker-compose up -d postgres
	@sleep 2
	@echo "✅ PostgreSQL is running on localhost:5432"

postgres-down:
	@echo "🛑 Stopping PostgreSQL..."
	@docker-compose down

postgres-status:
	@docker-compose ps postgres

# Start local development environment
start-local: postgres-up
	@echo ""
	@echo "📝 Next steps:"
	@echo ""
	@echo "1. Backend (in a new terminal):"
	@echo "   cd backend"
	@echo "   make migrate-up  # Run migrations (first time)"
	@echo "   make run         # Start backend server"
	@echo ""
	@echo "2. Frontend (in another terminal):"
	@echo "   cd frontend"
	@echo "   npm install      # If not already done"
	@echo "   npm run dev      # Start frontend"
	@echo ""
	@echo "Backend will run on: http://localhost:8080"
	@echo "Frontend will run on: http://localhost:5173"
	@echo ""
	@echo "To stop PostgreSQL: make postgres-down"

# Install required development tools
install-tools:
	@echo "📦 Installing required development tools..."
	@echo "Installing golang-migrate..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Installing sqlc..."
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "Installing gqlgen..."
	@go install github.com/99designs/gqlgen@latest
	@echo ""
	@echo "✅ Tools installed!"
	@echo "💡 Make sure $(shell go env GOPATH)/bin is in your PATH"
	@echo "   Add this to your ~/.bashrc or ~/.zshrc:"
	@echo "   export PATH=\"$$PATH:$(shell go env GOPATH)/bin\""

# Backend targets
backend-install:
	@echo "📦 Installing backend dependencies..."
	@cd backend && make deps

backend-migrate:
	@echo "🔄 Running database migrations..."
	@cd backend && make migrate-up

backend-run:
	@echo "🚀 Starting backend server..."
	@cd backend && make run

# Frontend targets
frontend-install:
	@echo "📦 Installing frontend dependencies..."
	@cd frontend && npm install

frontend-dev:
	@echo "🚀 Starting frontend dev server..."
	@cd frontend && npm run dev

