BIN := bin/api

run:
	go run ./cmd/parking

build:
	go build -trimpath -o $(BIN) ./cmd/parking

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
	go mod verify

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

migration:
	@if [ -z "$(name)" ]; then echo "Usage: make migration name=<migration_name>"; exit 1; fi
	@echo "\033[32mCreating migration files\033[39m"
	touch ./migrations/`date +%s`_$(name).up.sql
	touch ./migrations/`date +%s`_$(name).down.sql
