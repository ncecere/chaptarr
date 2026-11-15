VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build web run dev test clean

# One self-contained binary with the UI embedded.
build: web
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/chaptarr ./cmd/chaptarr

web:
	cd web && pnpm install --frozen-lockfile && pnpm build

# Build and run with the database in ./data.
run: build
	./bin/chaptarr -data ./data

# Backend only, for use with `cd web && pnpm dev` (Vite proxies /api to :8789).
dev:
	go run ./cmd/chaptarr -data ./data -debug

test:
	go vet ./...
	go test ./...
	cd web && pnpm lint

clean:
	rm -rf bin web/dist
