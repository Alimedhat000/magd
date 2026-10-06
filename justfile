# Build the magd binary into ./bin
build:
    go build -o bin/magd ./cmd/magd

# Build, then run. Pass a file to execute it: `just run script.magd`.
# With no arguments, starts the REPL.
run *args: build
    ./bin/magd {{args}}

# Run all tests
test *args:
    go test ./... {{args}}

# Run tests with coverage, report per-package
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1

# Vet, lint, and format-check. Run this before pushing.
check:
    go vet ./...
    golangci-lint run ./...
    gofmt -l .

# Apply formatter and safe linter autofixes
fmt:
    gofmt -w .
    golangci-lint run --fix ./...

# Add or tidy dependencies
tidy:
    go mod tidy

# Remove build artifacts
clean:
    go clean
