.PHONY: format lint check test race build

format:
	golangci-lint fmt

lint:
	golangci-lint run

check: lint
	golangci-lint fmt --diff
	go vet ./...
	go tool staticcheck ./...

test:
	go test ./...

race:
	go test -race ./...

build:
	go build ./...
