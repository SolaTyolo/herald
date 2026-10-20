.PHONY: dev test build run-api run-worker playground docker-up docker-down docker-app

DATABASE_URL ?= postgres://herald:herald@localhost:5432/herald?sslmode=disable
REDIS_ADDR ?= localhost:6379
HTTP_ADDR ?= :8080
DEV_MODE ?= true
API_KEY ?= herald-dev-key
STORE_TYPE ?= db
DB_DRIVER ?= postgres
STORE_FILE_PATH ?= ./data

PROVIDERS_CONFIG ?= ./config/providers.yaml

export DATABASE_URL REDIS_ADDR HTTP_ADDR DEV_MODE API_KEY
export STORE_TYPE DB_DRIVER STORE_FILE_PATH PROVIDERS_CONFIG

COMPOSE := docker compose -f deploy/docker-compose.yml --env-file .env

docker-up:
	$(COMPOSE) up -d

docker-down:
	$(COMPOSE) --profile app down

docker-app:
	$(COMPOSE) --profile app up -d --build

build:
	go build -o bin/herald-api ./cmd/api
	go build -o bin/herald-worker ./cmd/worker

run-api: build
	./bin/herald-api

run-worker: build
	./bin/herald-worker

playground:
	go run ./e2e/playground

test:
	go test ./... -count=1

dev: docker-up
	@echo "Waiting for postgres..."
	@sleep 3
	$(MAKE) build
	@echo "Run: make run-api & make run-worker"
