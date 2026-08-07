.PHONY: proto sqlc run dev migrate

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/auth/auth.proto

sqlc:
	sqlc generate

run:
	go run cmd/server/main.go

dev:
	air

migrate:
	psql $$DATABASE_URL -f internal/db/migrations/000001_init.up.sql
