# Binary name
BINARY_NAME=zee-mirror

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

.PHONY: all build clean test lint fmt tidy run help

all: lint test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

lint:
	golangci-lint run ./...

fmt:
	$(GOCMD) fmt ./...

tidy:
	$(GOMOD) tidy

run: build
	./$(BINARY_NAME)

help:
	@echo "Available commands:"
	@echo "  make build  - Build the binary"
	@echo "  make clean  - Remove binary"
	@echo "  make test   - Run tests"
	@echo "  make lint   - Run golangci-lint"
	@echo "  make fmt    - Format code"
	@echo "  make tidy   - Tidy go modules"
	@echo "  make run    - Build and run"
	@echo "  make all    - Run lint, test and build"
