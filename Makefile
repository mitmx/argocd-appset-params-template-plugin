BINARY := bin/argocd-appset-params-template-plugin

.PHONY: fmt check-fmt vet test build run clean image

fmt:
	gofmt -w ./cmd ./internal

check-fmt:
	@test -z "$$(gofmt -l ./cmd ./internal)" || (gofmt -l ./cmd ./internal && exit 1)

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server

run:
	PLUGIN_TOKEN=dev-token go run ./cmd/server

image:
	docker build -t argocd-appset-params-template-plugin:dev .

clean:
	rm -rf bin dist coverage.out *.coverprofile
