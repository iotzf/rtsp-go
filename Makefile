# RTSP Go项目 Makefile

.PHONY: all build clean test server client examples run-server run-client

# 默认目标
all: build

# 构建所有组件
build: server client examples

# 构建服务端
server:
	@echo "构建RTSP服务端..."
	cd server && go build -o rtsp-server server.go
	@echo "服务端构建完成: server/rtsp-server"

# 构建客户端
client:
	@echo "构建RTSP客户端..."
	cd client && go build -o rtsp-client client.go
	@echo "客户端构建完成: client/rtsp-client"

# 构建示例程序
examples:
	@echo "构建示例程序..."
	go build -o examples examples.go
	@echo "示例程序构建完成: examples"

# 运行示例程序
test: examples
	@echo "运行示例程序..."
	./examples

# 运行服务端
run-server: server
	@echo "启动RTSP服务端..."
	cd server && ./rtsp-server

# 运行客户端
run-client: client
	@echo "启动RTSP客户端..."
	cd client && ./rtsp-client

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -f examples
	rm -f server/rtsp-server
	rm -f client/rtsp-client
	@echo "清理完成"

# 安装依赖
deps:
	@echo "检查依赖..."
	go mod tidy
	go mod verify

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	go vet ./...

# 运行测试
check: fmt lint test

# 帮助信息
help:
	@echo "RTSP Go项目构建系统"
	@echo ""
	@echo "可用目标:"
	@echo "  all         - 构建所有组件 (默认)"
	@echo "  build       - 构建所有组件"
	@echo "  server      - 构建RTSP服务端"
	@echo "  client      - 构建RTSP客户端"
	@echo "  examples    - 构建示例程序"
	@echo "  test        - 运行示例程序测试"
	@echo "  run-server  - 启动RTSP服务端"
	@echo "  run-client  - 启动RTSP客户端"
	@echo "  clean       - 清理构建文件"
	@echo "  deps        - 检查并安装依赖"
	@echo "  fmt         - 格式化代码"
	@echo "  lint        - 代码检查"
	@echo "  check       - 运行所有检查"
	@echo "  help        - 显示此帮助信息"
	@echo ""
	@echo "使用示例:"
	@echo "  make build     # 构建所有组件"
	@echo "  make run-server # 启动服务端"
	@echo "  make run-client # 启动客户端"
	@echo "  make clean     # 清理文件"
