.PHONY: build run test dev

build:
	go build -o bin/upal ./cmd/upal

run: build
	./bin/upal

test:
	go test ./... -v -race

dev:
	go run ./cmd/upal
