.DEFAULT_GOAL := help
TEST_COMPOSE = docker compose -p car-service-tests -f deployment/docker/docker-compose.test.yml
SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
SWAG = go run github.com/swaggo/swag/cmd/swag@v1.8.1

.PHONY: help run build fmt lint test test-unit test-http race test-up test-load test-integration test-down migrate-up migrate-down generate docs
help:
	@echo 'run / build              Run the API / build bin/car_service'
	@echo 'fmt / lint               Format Go / check formatting and go vet'
	@echo 'test / test-unit / race  Docker-free tests / same / race detector'
	@echo 'test-http                HTTP status/header/body contracts, no Docker'
	@echo 'test-load                Isolated PostgreSQL, bounded load scenarios'
	@echo 'test-integration         Isolated PostgreSQL, all integration and load tests with -race'
	@echo 'test-down                Stop only the isolated test stack'
	@echo 'migrate-up               Apply local Compose database migrations'
	@echo 'migrate-down             Roll back ONE migration (requires CONFIRM=drop-listing-details)'
	@echo 'generate / docs          Regenerate sqlc / Swagger with pinned Go tools'

run:
	go run ./cmd/car_service
build:
	go build -o bin/car_service ./cmd/car_service
fmt:
	gofmt -w cmd internal test
lint:
	@test -z "$$(gofmt -l cmd internal test)" || (echo 'Run make fmt'; exit 1)
	go vet ./...
test: test-unit
test-unit:
	go test -count=1 -timeout=2m ./...
test-http:
	go test -count=1 -timeout=2m ./internal/handler -run '^Test(API|HTTP)'
race:
	go test -race -count=1 -timeout=2m ./...
test-up:
	$(TEST_COMPOSE) up -d --wait --wait-timeout 120 postgres
	$(TEST_COMPOSE) run --rm migrate
test-integration: test-up
	CAR_SERVICE_BASE_URL= go test -race -tags=integration,load -count=1 -timeout=3m ./...
test-load: test-up
	CAR_SERVICE_BASE_URL= go test -tags=integration,load ./test -run '^TestLoad' -count=1 -timeout=2m -v
test-down:
	$(TEST_COMPOSE) down --remove-orphans
migrate-up:
	docker compose run --rm migrate
migrate-down:
	@test "$(CONFIRM)" = 'drop-listing-details' || (echo 'Rollback deletes the three new fields. Back up first. Use CONFIRM=drop-listing-details'; exit 1)
	@test "$$(docker compose exec -T postgres psql -U postgres -d cars -Atc 'SELECT version FROM schema_migrations WHERE NOT dirty')" = '2' || (echo 'Refusing rollback: expected clean migration version 2. Inspect migration status first.'; exit 1)
	docker compose run --rm migrate -path=/migrations -database='postgres://postgres:postgres@postgres:5432/cars?sslmode=disable' down 1
generate:
	$(SQLC) generate
docs:
	$(SWAG) init -g cmd/car_service/main.go -o docs
