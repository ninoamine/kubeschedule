MODULE := github.com/ninoamine/kubeschedule
BIN := bin/kubeschedule

GO := go


.PHONY: all build test lint clean


all: lint test build


build:
	@echo "Building $(BIN)..."
	$(GO) build -o $(BIN) ./cmd/...


test:
	@echo "Running tests..."
	$(GO) test -race -coverprofile=coverage.out ./...

lint:
	@echo "Running lint..."
	golangci-lint run ./...


clean:
	@echo "Cleaning up..."
	rm -rf bin/ coverage.out
