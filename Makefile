.PHONY: all run build test clean

BINARY_NAME := .tmp_heimdall

all: build

## build: Compile heimdall binary
build:
	@echo "🔨 Compiling heimdall..."
	@mkdir -p bin
	@go build -o bin/heimdall .

## run: Build and run heimdall, auto-cleanup temporary binary on exit (Ctrl+C)
run:
	@go build -o $(BINARY_NAME) .
	@trap 'echo "\n🧹 Cleaning up temporary binary..."; rm -f $(BINARY_NAME); exit 0' INT TERM EXIT; \
	./$(BINARY_NAME) $(ARGS)

## test: Run all unit tests
test:
	@echo "🧪 Running unit tests..."
	@go test -v ./...

## clean: Remove build artifacts and binaries
clean:
	@echo "🧹 Removing binaries..."
	@rm -f $(BINARY_NAME) bin/heimdall agent-observer
