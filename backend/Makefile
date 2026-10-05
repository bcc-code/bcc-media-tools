dev:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

# Usage: make migration name=add_foo
migration:
	go run github.com/pressly/goose/v3/cmd/goose@v3.28.0 -dir editorial/migrations -s create $(name) sql
