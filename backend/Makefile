include ../.env
export

GOOSE_DRIVER := postgres
GOOSE_DBSTRING := host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)
GOOSE_MIGRATION_DIR := migrations

.PHONY: run migrate-up migrate-down migrate-status

run:
	go run ./cmd/api

migrate-up:
	goose up

migrate-down:
	goose down

migrate-status:
	goose status