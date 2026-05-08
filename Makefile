MODULE := github.com/ninoamine/kubeschedule
BIN := bin/kubeschedule

GO := go


.PHONY: all build test lint clean generate


all: lint test build


generate:
	@echo "Generating DeepCopy and CRD manifests..."
	go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0 object crd paths=./api/v1alpha1/... output:crd:dir=./config/crd


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
