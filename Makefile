.PHONY: dev test build run-api run-worker docker-up docker-down

DATABASE_URL ?= postgres://herald:herald@localhost:5432/herald?sslmode=disable
REDIS_ADDR ?= localhost:6379
ENCRYPTION_KEY ?= 0123456789abcdef0123456789abcdef
HTTP_ADDR ?= :8080
DEV_MODE ?= true
STORE_TYPE ?= db
DB_DRIVER ?= postgres
STORE_FILE_PATH ?= ./data

export DATABASE_URL REDIS_ADDR ENCRYPTION_KEY HTTP_ADDR DEV_MODE
export STORE_TYPE DB_DRIVER STORE_FILE_PATH

docker-up:
	docker compose -f deploy/docker-compose.yml up -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

build:
	go build -o bin/herald-api ./cmd/api
	go build -o bin/herald-worker ./cmd/worker
	go build -o bin/rabbit-inapp-bridge ./cmd/rabbit-inapp-bridge

run-api: build
	./bin/herald-api

run-worker: build
	./bin/herald-worker

test:
	go test ./... -count=1

dev: docker-up
	@echo "Waiting for postgres..."
	@sleep 3
	$(MAKE) build
	@echo "Run: make run-api & make run-worker"
