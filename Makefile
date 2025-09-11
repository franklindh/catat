URL = "postgresql://root:password@localhost:5432/catat_db?sslmode=disable"

postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:17-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root catat_db

dropdb:
	docker exec -it postgres dropdb catat_db

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

server:
	go run main.go

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test test_coverage server