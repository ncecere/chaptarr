VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build web run dev test clean release-check

# One self-contained binary with the UI embedded.
build: web
	go build -ldflags "$(LDFLAGS)" -o bin/chaptarr ./cmd/chaptarr

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

# Before tagging: `make release-check TAG=v0.2.0` runs the tests and the
# checks CI applies to a release tag (README, "Releasing").
release-check: test
	TAG=$(TAG) TAG_COMMIT=HEAD deploy/release/check-tag.sh

clean:
	rm -rf bin dist web/dist
