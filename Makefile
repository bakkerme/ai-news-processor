.PHONY: run run-benchmark build test clean help

# Default target
help:
	@echo "Available targets:"
	@echo "  run           - Run main application with LocalLLaMa persona"
	@echo "  run-benchmark - Run benchmark application"
	@echo "  build         - Build both applications"
	@echo "  test          - Run all tests"
	@echo "  clean         - Clean build artifacts"

# Run main application (equivalent to "Launch Package" config)
run:
	go run ./ -persona LocalLLaMa

# Run benchmark application (equivalent to "Launch Benchmark" config)
run-benchmark:
	cd benchmark && \
	go run ./

# Build applications
build:
	go build -o ai-news-processor main.go
	go build -o benchmark/benchmark benchmark/main.go

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f ai-news-processor
	rm -f benchmark/benchmark

# Development commands from CLAUDE.md
dev-all:
	go run main.go --persona=all

dev-direct:
	go run ./internal

# Dependency management
deps:
	go mod download
	go mod verify

deps-update:
	go mod tidy
