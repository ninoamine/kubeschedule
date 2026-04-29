MODULE := github.com/ninoamine/kubeschedule
BIN := bin/kubeschedule

GO := go


.PHONY: all build test lint clean


all: lint test build


build:
	@echo "Building binaries..."
	$(GO) build -o bin/kubeschedule ./cmd/kubeschedule
	$(GO) build -o bin/explore ./cmd/explore


test:
	@echo "Running tests..."
	$(GO) test -race -coverprofile=coverage.out ./...

lint:
	@echo "Running lint..."
	golangci-lint run ./...


clean:
	@echo "Cleaning up..."
	rm -rf bin/ coverage.out
