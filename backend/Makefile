include ../.env
export

GOOSE_DRIVER := postgres
GOOSE_DBSTRING := host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)
GOOSE_MIGRATION_DIR := migrations

.PHONY: run seed check migrate-up migrate-down migrate-status

run:
	go run ./cmd/api

# Crea il primo admin; la password viene chiesta su stdin.
# Esempio: make seed ARGS="-username admin -email admin@example.com -first-name Mario -last-name Rossi -birth-date 1980-01-31"
seed:
	go run ./cmd/seed $(ARGS)

check:
	gofmt -l . | (! grep .)
	go vet ./...
	go test ./...

migrate-up:
	goose up

migrate-down:
	goose down

migrate-status:
	goose status