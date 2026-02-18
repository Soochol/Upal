.PHONY: build run test dev build-frontend dev-frontend dev-backend

build-frontend:
	cd web && npx vite build

build: build-frontend
	go build -o bin/upal ./cmd/upal

run: build
	./bin/upal serve

test:
	go test ./... -v -race

test-frontend:
	cd web && npx tsc -b

dev-frontend:
	cd web && npm run dev

dev-backend:
	go run ./cmd/upal serve

dev:
	@echo "Run 'make dev-backend' and 'make dev-frontend' in separate terminals"
