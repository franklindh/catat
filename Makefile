URL = "postgresql://root:password@localhost:5432/catat?sslmode=disable"

postgres:
	docker run --name postgres-catat -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:17-alpine

createdb:
	docker exec -it postgres-catat createdb --username=root --owner=root catat

dropdb:
	docker exec -it postgres-catat dropdb catat

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

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

dev:
	cd web && npm run dev

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/franklindh/catat/db/sqlc Store

swagger:
	swag init --parseDependency --parseInternal

.PHONY: postgres createdb dropdb new_migration migrateup migratedown sqlc test test_coverage server mock swagger