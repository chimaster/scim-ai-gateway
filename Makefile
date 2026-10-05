.PHONY: all build run test bench integration clean check-git

BINARY_NAME=bin/gateway
MAIN_PATH=./cmd/gateway/main.go

all: build

# Build the Linux native binary into bin/
build:
	@mkdir -p bin
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Run local development server
run: build
	./$(BINARY_NAME)

# Run unit tests across all packages
test:
	go test -v ./...

# Run performance benchmarks without maxing out local CPU
bench:
	go test -bench=. -benchmem -cpu=2 ./pkg/...

# Run end-to-end integration test against live running server
integration: build
	@chmod +x scripts/test_integration.sh
	@./scripts/test_integration.sh

# Clean compiled binaries and build artifacts
clean:
	rm -rf bin/
	rm -f main main.exe *.test

# Verify Git status is clean and binaries are ignored
check-git:
	git status
