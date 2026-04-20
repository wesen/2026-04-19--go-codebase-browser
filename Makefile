.PHONY: help dev-backend dev-frontend frontend-check frontend-build generate build smoke clean tidy test lint

BINARY := codebase-browser
PKG    := github.com/wesen/codebase-browser

help:
	@echo "Targets:"
	@echo "  dev-backend     Run Go server on :3001 (no embed, serves from disk)"
	@echo "  dev-frontend    Run Vite on :3000 with proxy to :3001"
	@echo "  frontend-check  TypeScript check"
	@echo "  frontend-build  Vite production build -> ui/dist/public/"
	@echo "  generate        Run go generate on the generator packages (builds index + copies assets)"
	@echo "  build           Build single embedded binary (tag: embed)"
	@echo "  smoke           Run binary and curl /api/index"
	@echo "  test            go test ./..."
	@echo "  lint            go vet ./..."
	@echo "  tidy            go mod tidy"

dev-backend:
	go run ./cmd/$(BINARY) serve --addr :3001

dev-frontend:
	pnpm -C ui run dev

frontend-check:
	pnpm -C ui run typecheck

frontend-build:
	pnpm -C ui run build

generate:
	go generate ./cmd/... ./internal/browser ./internal/docs ./internal/indexer ./internal/indexfs ./internal/server ./internal/sourcefs ./internal/web

build: generate
	go build -tags embed -o bin/$(BINARY) ./cmd/$(BINARY)

smoke: build
	./bin/$(BINARY) serve --addr :3001 &
	sleep 1
	curl -sf http://127.0.0.1:3001/api/index | head -c 200
	@echo
	pkill -f "bin/$(BINARY) serve" || true

test:
	go test ./... -count=1

lint:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin ui/dist internal/web/embed/public/* internal/sourcefs/embed/source/* internal/indexfs/embed/index.json
