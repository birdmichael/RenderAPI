# Makefile for RenderAPI

.PHONY: build test clean install examples test-coverage bench

# 默认目标
all: build

# 构建命令行工具
build:
	go build -o bin/httpclient ./cmd/httpclient

# 安装命令行工具
install:
	go install ./cmd/httpclient

# 运行测试
test:
	go test -v ./...

# 运行基础示例
example-basic:
	go run ./examples/basic/main.go

# 运行高级示例
example-advanced:
	go run ./examples/advanced/main.go

# 清理构建产物
clean:
	rm -rf bin/
	go clean

# 格式化代码
fmt:
	go fmt ./...

# 运行代码检查
lint:
	go vet ./...

# 生成文档
doc:
	godoc -http=:6060

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  make build           - 构建命令行工具"
	@echo "  make install         - 安装命令行工具"
	@echo "  make test            - 运行测试"
	@echo "  make example-basic   - 运行基础示例"
	@echo "  make example-advanced - 运行高级示例"
	@echo "  make clean           - 清理构建产物"
	@echo "  make fmt             - 格式化代码"
	@echo "  make lint            - 运行代码检查"
	@echo "  make doc             - 生成文档"
	@echo "  make test-coverage   - 运行测试并生成覆盖率报告"
	@echo "  make bench           - 运行基准测试"

.PHONY: test-coverage
test-coverage: ## 运行测试并生成覆盖率报告
	go test -v -cover -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: bench
bench: ## 运行基准测试
	go test -bench=. -benchmem ./... 