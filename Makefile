.PHONY: build build-worker run run-worker test migrate docker-up docker-down swagger

build:
	go build -o bin/api ./cmd/api

build-worker:
	go build -o bin/worker ./cmd/worker

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

air:
	air

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

migrate:
	@echo "Migrations are handled by GORM AutoMigrate on startup"

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

lint:
	golangci-lint run ./...
