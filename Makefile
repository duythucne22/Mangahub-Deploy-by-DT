.PHONY: help build-cli run-cli clean test all

help:
	@echo "MangaHub Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build-cli    - Build the CLI executable"
	@echo "  run-cli      - Run the CLI"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  all          - Build everything"

build-cli:
	@echo "Building MangaHub CLI..."
	go build -o bin/mangahub.exe ./cmd/cli
	@echo "✓ CLI built: bin/mangahub.exe"

run-cli:
	go run ./cmd/cli

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	@echo "✓ Clean complete"

test:
	go test ./...

all: build-cli
	@echo "✓ Build complete"
