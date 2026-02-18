# Local Development Guide

This guide explains how to run the application locally without Docker (except for PostgreSQL).

## Prerequisites

- Go 1.21 or later
- Node.js 18 or later
- Docker and Docker Compose (for PostgreSQL only)
- SQLC (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- golang-migrate (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)
- gqlgen (`go install github.com/99designs/gqlgen@latest`)

## Quick Start

### 1. Start PostgreSQL

```bash
make postgres-up
# or
make start-local  # Shows instructions after starting
```

This will start PostgreSQL on `localhost:5432` with:
- User: `postgres`
- Password: `postgres`
- Database: `finance_manager`

You can also use the convenience Makefile targets:
- `make start-local` - Start PostgreSQL and show setup instructions
- `make postgres-up` - Start PostgreSQL
- `make postgres-down` - Stop PostgreSQL
- `make postgres-status` - Check PostgreSQL status

### 2. Set Up Backend

```bash
cd backend

# Install dependencies
make deps

# Set up environment variables (optional - defaults work for local dev)
# Create a .env file or export:
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/finance_manager?sslmode=disable
export PORT=8080
export ENVIRONMENT=development
export JWT_SECRET=your-secret-key-change-in-production

# Run database migrations
make migrate-up

# Generate GraphQL code (if needed)
make generate

# Run the backend
make run
```

The backend will be available at `http://localhost:8080`

### 3. Set Up Frontend

```bash
cd frontend

# Install dependencies
npm install

# Set up environment variables (optional)
# Create a .env file or export:
export VITE_BACKEND_URL=http://localhost:8080

# Run the frontend
npm run dev
```

The frontend will be available at `http://localhost:5173`

## Environment Variables

### Backend

Create a `.env` file in the `backend/` directory or export environment variables:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/finance_manager?sslmode=disable
PORT=8080
ENVIRONMENT=development
JWT_SECRET=your-secret-key-change-in-production
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
```

### Frontend

Create a `.env` file in the `frontend/` directory or export environment variables:

```env
VITE_BACKEND_URL=http://localhost:8080
```

## Database Migrations

```bash
cd backend

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Code Generation

### SQLC (after modifying SQL queries)

```bash
cd backend
make sqlc
```

### GraphQL (after modifying schema.graphqls)

```bash
cd backend
make generate
```

## Stopping Services

```bash
# Stop PostgreSQL
make postgres-down

# Or stop and remove volumes
docker-compose down -v
```

## Using Makefile

The root Makefile provides convenient targets:

```bash
# Show all available targets
make help

# PostgreSQL management
make postgres-up        # Start PostgreSQL
make postgres-down      # Stop PostgreSQL
make postgres-status    # Check status

# Backend
make backend-install    # Install dependencies
make backend-migrate    # Run migrations
make backend-run        # Start server

# Frontend
make frontend-install   # Install dependencies
make frontend-dev       # Start dev server
```

## Troubleshooting

### Backend can't connect to database

- Ensure PostgreSQL is running: `docker-compose ps`
- Check DATABASE_URL matches docker-compose settings
- Verify port 5432 is not blocked

### CORS errors

- Ensure `VITE_BACKEND_URL` is set correctly in frontend
- Check backend CORS configuration allows your frontend origin
- In development, backend allows all localhost origins automatically

### Port already in use

- Change `PORT` environment variable for backend
- Change port in `vite.config.ts` for frontend

