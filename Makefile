.PHONY: run build test lint docker-up docker-down docker-logs tidy

BIN := bin/api

## run: start the API server (requires a running Postgres on localhost:5432)
run:
	go run ./cmd/api

## build: compile the binary to ./bin/api
build:
	go build -trimpath -o $(BIN) ./cmd/api

## test: run all tests with race detector
test:
	go test -race -count=1 ./...

## lint: run golangci-lint (requires golangci-lint in PATH)
lint:
	golangci-lint run ./...

## tidy: tidy and verify modules
tidy:
	go mod tidy
	go mod verify

## docker-up: bring up Postgres + API via Docker Compose
docker-up:
	docker compose up --build

## docker-down: tear down Docker Compose stack and volumes
docker-down:
	docker compose down -v

## docker-logs: tail API container logs
docker-logs:
	docker compose logs -f api

## migration: create a new migration pair (use name=<migration_name>)
##   example: make migration name=add_indexes
.PHONY: migration
migration:
	@if [ -z "$(name)" ]; then echo "Usage: make migration name=<migration_name>"; exit 1; fi
	@echo "\033[32mCreating migration files\033[39m"
	touch ./migrations/`date +%s`_$(name).up.sql
	touch ./migrations/`date +%s`_$(name).down.sql
