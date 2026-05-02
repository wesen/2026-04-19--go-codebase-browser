.PHONY: help frontend-check frontend-build generate build smoke clean tidy test lint

BINARY := codebase-browser
PKG    := github.com/wesen/codebase-browser

help:
	@echo "Targets:"
	@echo "  frontend-check  TypeScript check"
	@echo "  frontend-build  Vite production build -> ui/dist/public/"
	@echo "  generate        Run go generate on the generator packages (builds index + copies assets)"
	@echo "  build           Build single embedded binary (tag: embed)"
	@echo "  smoke           Build the CLI"
	@echo "  test            go test ./..."
	@echo "  lint            go vet ./..."
	@echo "  tidy            go mod tidy"


frontend-check:
	pnpm -C ui run typecheck

frontend-build:
	pnpm -C ui run build

generate:
	go generate ./cmd/... ./internal/browser ./internal/docs ./internal/indexer ./internal/indexfs ./internal/sourcefs


build: generate
	go build -o bin/$(BINARY) ./cmd/$(BINARY)

smoke: build
	./bin/$(BINARY) --help >/dev/null
	@echo "smoke ok"

test:
	go test ./... -count=1

lint:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin ui/dist internal/sourcefs/embed/source/* internal/indexfs/embed/index.json
