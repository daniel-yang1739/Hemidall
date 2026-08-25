.PHONY: all run build test clean

BINARY_NAME := .tmp_agent_observer

all: build

## build: Compile agent-observer binary
build:
	@echo "🔨 Compiling agent-observer..."
	@go build -o $(BINARY_NAME) .

## run: Build and run observer, auto-cleanup binary on exit (Ctrl+C)
run: build
	@trap 'echo "\n🧹 Cleaning up temporary binary..."; rm -f $(BINARY_NAME); exit 0' INT TERM EXIT; \
	./$(BINARY_NAME) $(ARGS)

## test: Run all unit tests
test:
	@echo "🧪 Running unit tests..."
	@go test -v ./...

## clean: Remove build artifacts and binaries
clean:
	@echo "🧹 Removing binaries..."
	@rm -f $(BINARY_NAME) agent-observer
