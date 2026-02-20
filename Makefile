.PHONY: build run test dev build-frontend dev-frontend dev-backend test-zimage-mock

build-frontend:
	cd web && npm run build

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

test-zimage-mock:
	@echo "Starting Z-IMAGE mock server..."
	@python3 scripts/zimage_server.py --mock --mock-delay 0 --port 8090 & \
		PID=$$!; \
		sleep 2; \
		echo "Health check:"; \
		curl -s http://localhost:8090/health | python3 -m json.tool; \
		echo "Generate test:"; \
		curl -s -X POST http://localhost:8090/generate \
			-H "Content-Type: application/json" \
			-d '{"prompt":"test"}' | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'OK: {len(d[\"image\"])} chars base64, mime={d[\"mime_type\"]}')"; \
		kill $$PID 2>/dev/null; \
		echo "Done."
