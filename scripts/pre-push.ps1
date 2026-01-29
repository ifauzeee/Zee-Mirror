Write-Host "Running pre-push quality checks..." -ForegroundColor Cyan

# Run linting
Write-Host "Running linter..." -ForegroundColor Yellow
golangci-lint run ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Linter failed. Push aborted." -ForegroundColor Red
    exit 1
}

# Run tests
Write-Host "Running tests..." -ForegroundColor Yellow
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed. Push aborted." -ForegroundColor Red
    exit 1
}

Write-Host "All checks passed. Proceeding with push." -ForegroundColor Green
exit 0
