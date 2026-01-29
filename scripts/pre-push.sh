#!/bin/sh

echo "Running pre-push quality checks..."

# Run linting
echo "Running linter..."
golangci-lint run ./...
if [ $? -ne 0 ]; then
    echo "Linter failed. Push aborted."
    exit 1
fi

# Run tests
echo "Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "Tests failed. Push aborted."
    exit 1
fi

echo "All checks passed. Proceeding with push."
exit 0
