.PHONY: dev start migrate sqlc db-ensure db-reset

# Default database URL
DATABASE_URL ?= postgres://finance:finance_dev@localhost:5432/finance_tracker?sslmode=disable

# Admin connection for bootstrapping (matches docker-compose defaults)
ADMIN_URL ?= postgres://admin:root@localhost:5432/postgres?sslmode=disable

COMPOSE_FILE := $(shell git rev-parse --show-toplevel)/docker-compose.yml

# Full dev startup: ensure DB exists, migrate, then run with hot-reload
dev: db-ensure migrate start

# Create the finance user and database if they don't already exist
db-ensure:
	@echo "Ensuring database and user exist..."
	@psql "$(ADMIN_URL)" -tc "SELECT 1 FROM pg_roles WHERE rolname='finance'" | grep -q 1 \
		|| psql "$(ADMIN_URL)" -c "CREATE USER finance WITH PASSWORD 'finance_dev';"
	@psql "$(ADMIN_URL)" -tc "SELECT 1 FROM pg_catalog.pg_database WHERE datname='finance_tracker'" | grep -q 1 \
		|| psql "$(ADMIN_URL)" -c "CREATE DATABASE finance_tracker OWNER finance;"
	@echo "Database ready."

# Destroy and recreate the postgres volume so init scripts run fresh
db-reset:
	docker compose -f "$(COMPOSE_FILE)" down -v
	docker compose -f "$(COMPOSE_FILE)" up -d
	@echo "Waiting for postgres to be healthy..."
	@until docker compose -f "$(COMPOSE_FILE)" ps postgres --format json | grep -q '"healthy"'; do sleep 1; done
	@echo "Postgres is ready."

start:
	air

migrate:
	goose -dir internal/db/migrations postgres "$(DATABASE_URL)" up

sqlc:
	sqlc generate
