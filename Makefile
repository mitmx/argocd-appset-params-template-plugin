.PHONY: test build run

test:
	go test ./...

build:
	go build ./cmd/server

run:
	PLUGIN_TOKEN=dev-token go run ./cmd/server
