.PHONY: build run test dev dev-frontend dev-backend

build:
	go build -o bin/upal ./cmd/upal

run: build
	./bin/upal

test:
	go test ./... -v -race

dev-frontend:
	cd web && npm run dev

dev-backend:
	go run ./cmd/upal serve

dev:
	@echo "Run 'make dev-backend' and 'make dev-frontend' in separate terminals"
