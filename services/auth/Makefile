.PHONY: sqlc run dev migrate proto

# Contracts live in libs/proto/auth/v1 and are generated repo-wide with buf.
proto:
	cd ../.. && just proto

sqlc:
	sqlc generate

run:
	go run cmd/server/main.go

dev:
	air

migrate:
	psql $$DATABASE_URL -f internal/db/migrations/000001_init.up.sql
