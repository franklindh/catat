-include .env
export

DB_USER ?= root
DB_PASSWORD ?= password
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_NAME ?= catat_db
DB_SSLMODE ?= disable


URL = "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
TEST_URL = "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/test_$(DB_NAME)?sslmode=$(DB_SSLMODE)"

postgres:
	docker run --name postgres -p $(DB_PORT):$(DB_PORT) -e POSTGRES_USER=$(DB_USER) -e POSTGRES_PASSWORD=$(DB_PASSWORD) -d postgres:17-alpine

createdb:
	docker exec -it postgres createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME)

dropdb:
	docker exec -it postgres dropdb $(DB_NAME)

migrateup:
	migrate -path db/migration/ -database $(URL) -verbose up

migratedown:
	migrate -path db/migration/ -database $(URL) -verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

test_coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test test_coverage createtestdb droptestdb migrateup_test migratedown_test