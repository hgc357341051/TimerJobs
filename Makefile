# 小胡专用定时任务系统 Makefile

# 变量定义
APP_NAME = jobs
VERSION = 1.2.0
BUILD_TIME = $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go相关变量
GO = go
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# 构建目录
BUILD_DIR = build
DIST_DIR = dist

# 构建标签
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w"

.PHONY: help build clean test run install uninstall start stop reload lint fmt vet daemon daemon-start daemon-stop daemon-status

# 默认目标
help:
	@echo "小胡专用定时任务系统 - 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build         - 构建可执行文件"
	@echo "  build-all     - 构建所有平台的可执行文件"
	@echo "  clean         - 清理构建文件"
	@echo "  test          - 运行测试"
	@echo "  run           - 前台模式运行程序"
	@echo "  start         - 前台模式启动"
	@echo "  start-bg      - 后台模式启动"
	@echo "  start-daemon  - 守护模式启动"
	@echo "  stop          - 停止后台模式"
	@echo "  stop-all      - 停止守护模式(所有进程)"
	@echo "  status        - 查看运行状态"
	@echo "  install       - 安装为系统服务"
	@echo "  uninstall     - 卸载系统服务"
	@echo "  reload        - 重载配置"
	@echo "  lint          - 代码检查"
	@echo "  fmt           - 代码格式化"
	@echo "  vet           - 代码静态分析"
	@echo "  docker        - 构建Docker镜像"
	@echo "  release       - 创建发布包"
	@echo ""
	@echo "使用示例:"
	@echo "  make start        # 前台模式运行"
	@echo "  make start-bg     # 后台模式运行"
	@echo "  make start-daemon # 守护模式运行"
	@echo "  make stop         # 停止后台进程"
	@echo "  make stop-all     # 停止所有进程"
	@echo "  make status       # 查看状态"

# 构建可执行文件
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) main.go
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 构建所有平台的可执行文件
build-all:
	@echo "构建所有平台的可执行文件..."
	@mkdir -p $(DIST_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 main.go
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe main.go
	GOOS=windows GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe main.go
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 main.go
	
	@echo "构建完成，文件位于 $(DIST_DIR)/"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean -cache
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 运行程序
run:
	@echo "前台模式运行程序..."
	$(GO) run main.go

# 安装为系统服务
install: build
	@echo "安装系统服务..."
	./$(BUILD_DIR)/$(APP_NAME) install

# 卸载系统服务
uninstall: build
	@echo "卸载系统服务..."
	./$(BUILD_DIR)/$(APP_NAME) uninstall

# 前台模式启动
start: build
	@echo "前台模式启动..."
	./$(BUILD_DIR)/$(APP_NAME) start

# 后台模式启动
start-bg: build
	@echo "后台模式启动..."
	./$(BUILD_DIR)/$(APP_NAME) start -d

# 守护模式启动
start-daemon: build
	@echo "守护模式启动..."
	./$(BUILD_DIR)/$(APP_NAME) start -d -f

# 停止后台模式
stop: build
	@echo "停止后台模式..."
	./$(BUILD_DIR)/$(APP_NAME) stop

# 停止守护模式(所有进程)
stop-all: build
	@echo "停止守护模式(所有进程)..."
	./$(BUILD_DIR)/$(APP_NAME) stop -f

# 查看运行状态
status: build
	@echo "查看运行状态..."
	./$(BUILD_DIR)/$(APP_NAME) status

# 重载配置
reload: build
	@echo "重载配置..."
	./$(BUILD_DIR)/$(APP_NAME) reload

# 代码检查
lint:
	@echo "代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码检查"; \
	fi

# 代码格式化
fmt:
	@echo "代码格式化..."
	$(GO) fmt ./...

# 代码静态分析
vet:
	@echo "代码静态分析..."
	$(GO) vet ./...

# 更新依赖
deps:
	@echo "更新依赖..."
	$(GO) mod tidy
	$(GO) mod download

# 生成API文档
docs:
	@echo "生成API文档..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g main.go -o docs; \
	else \
		echo "swag 未安装，请运行: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# 构建Docker镜像
docker:
	@echo "构建Docker镜像..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# 创建发布包
release: build-all
	@echo "创建发布包..."
	@mkdir -p $(DIST_DIR)/release
	@cp -r config $(DIST_DIR)/release/
	@cp README.md $(DIST_DIR)/release/
	@cp Makefile $(DIST_DIR)/release/
	@cd $(DIST_DIR) && tar -czf release-$(VERSION).tar.gz release/ $(APP_NAME)-*
	@echo "发布包创建完成: $(DIST_DIR)/release-$(VERSION).tar.gz"

# 开发模式（使用air热重载）
dev:
	@echo "启动开发模式..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air 未安装，请运行: go install github.com/cosmtrek/air@latest"; \
		echo "或者直接运行: go run main.go"; \
	fi

# 性能分析
profile:
	@echo "启动性能分析..."
	$(GO) run main.go &
	@echo "程序已启动，访问 http://localhost:8080/debug/pprof/ 查看性能数据"
	@echo "按 Ctrl+C 停止程序"

# 检查代码覆盖率
coverage:
	@echo "检查代码覆盖率..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 安全扫描
security:
	@echo "安全扫描..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec 未安装，请运行: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# 显示版本信息
version:
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"
	@echo "Go版本: $(shell go version)"
	@echo "操作系统: $(GOOS)/$(GOARCH)"

# MCP服务器相关命令
mcp-build:
	@echo "构建MCP服务器..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/mcp.exe mcp/main.go
	@echo "MCP服务器构建完成: $(BUILD_DIR)/mcp.exe"

mcp-run:
	@echo "启动MCP服务器..."
	$(GO) run mcp/main.go -port 8081

mcp-test:
	@echo "测试MCP服务器..."
	@if command -v python >/dev/null 2>&1; then \
		python mcp/test_mcp.py; \
	else \
		echo "Python 未安装，跳过测试"; \
	fi

mcp-start:
	@echo "启动MCP服务器..."
	./$(BUILD_DIR)/mcp.exe -port 8081

# 一键构建并测试MCP
mcp-all: mcp-build mcp-test

# Go项目Makefile，支持Windows下sqlite3自动编译

BINARY_NAME = xiaohuAdmin.exe
SRC = main.go

.PHONY: all build clean

all: build

build:
	@echo "[INFO] 检查gcc..."
	@gcc --version >nul 2>&1 || (echo [ERROR] 未检测到gcc，请检查MinGW安装与环境变量！ & exit 1)
	@echo "[INFO] 开始编译..."
	@go build -o $(BINARY_NAME) $(SRC)
	@echo "[INFO] 编译完成，输出文件: $(BINARY_NAME)"

clean:
	@del /f /q $(BINARY_NAME) 2>nul || exit 0
	@echo "[INFO] 已清理编译产物"