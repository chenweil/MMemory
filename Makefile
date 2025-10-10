.PHONY: build clean run test docker-build docker-up docker-down help

# 变量定义
APP_NAME=mmemory
BIN_DIR=bin
CMD_DIR=cmd/bot
DOCKER_IMAGE=$(APP_NAME):latest

# 默认目标
.DEFAULT_GOAL := help

## build: 构建二进制文件到 bin/ 目录
build:
	@echo "🔨 构建 $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build -ldflags '-w -s' -o $(BIN_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "✅ 构建完成: $(BIN_DIR)/$(APP_NAME)"

## run: 运行应用程序
run:
	@echo "🚀 运行 $(APP_NAME)..."
	go run ./$(CMD_DIR)/main.go

## test: 运行所有测试
test:
	@echo "🧪 运行测试..."
	go test -v ./...

## test-cover: 运行测试并生成覆盖率报告
test-cover:
	@echo "📊 生成测试覆盖率报告..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

## clean: 清理构建产物
clean:
	@echo "🧹 清理构建产物..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	rm -f bot  # 清理根目录的bot文件
	@echo "✅ 清理完成"

## docker-build: 构建Docker镜像
docker-build:
	@echo "🐳 构建Docker镜像..."
	docker-compose build
	@echo "✅ Docker镜像构建完成"

## docker-up: 启动Docker容器
docker-up:
	@echo "🐳 启动Docker容器..."
	docker-compose up -d
	@echo "✅ Docker容器已启动"

## docker-down: 停止Docker容器
docker-down:
	@echo "🐳 停止Docker容器..."
	docker-compose down
	@echo "✅ Docker容器已停止"

## docker-logs: 查看Docker容器日志
docker-logs:
	docker-compose logs -f

## docker-rebuild: 重新构建并启动Docker容器
docker-rebuild:
	@echo "🐳 重新构建并启动Docker容器..."
	docker-compose down
	docker-compose up --build -d
	@echo "✅ Docker容器已重启"

## fmt: 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	go fmt ./...
	@echo "✅ 代码格式化完成"

## lint: 代码检查
lint:
	@echo "🔍 代码检查..."
	golangci-lint run
	@echo "✅ 代码检查完成"

## tidy: 整理依赖
tidy:
	@echo "📦 整理依赖..."
	go mod tidy
	@echo "✅ 依赖整理完成"

## help: 显示帮助信息
help:
	@echo "MMemory 项目构建工具"
	@echo ""
	@echo "使用方法:"
	@echo "  make [target]"
	@echo ""
	@echo "可用目标:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
