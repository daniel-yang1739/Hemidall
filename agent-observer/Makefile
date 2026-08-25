.PHONY: all run build test clean

BINARY_NAME := .tmp_agent_observer

all: build

## build: 編譯 agent-observer 二進制執行檔
build:
	@echo "🔨 Compiling agent-observer..."
	@go build -o $(BINARY_NAME) .

## run: 編譯並運行，結束時 (Ctrl+C) 自動清除臨時二進制檔
run: build
	@trap 'echo "\n🧹 Cleaning up temporary binary..."; rm -f $(BINARY_NAME); exit 0' INT TERM EXIT; \
	./$(BINARY_NAME) $(ARGS)

## test: 執行所有單元測試
test:
	@echo "🧪 Running unit tests..."
	@go test -v ./...

## clean: 清除編譯產出物
clean:
	@echo "🧹 Removing binaries..."
	@rm -f $(BINARY_NAME) agent-observer
