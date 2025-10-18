#!/bin/bash

# RTSP Go项目测试脚本

echo "RTSP Go项目测试脚本"
echo "=================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

echo "Go版本: $(go version)"

# 编译项目
echo ""
echo "编译项目..."

# 编译协议库
echo "编译协议库..."
cd rtspProtocon
go build -v
if [ $? -ne 0 ]; then
    echo "错误: 协议库编译失败"
    exit 1
fi
cd ..

# 编译服务端
echo "编译服务端..."
cd server
go build -o rtsp-server server.go
if [ $? -ne 0 ]; then
    echo "错误: 服务端编译失败"
    exit 1
fi
cd ..

# 编译客户端
echo "编译客户端..."
cd client
go build -o rtsp-client client.go
if [ $? -ne 0 ]; then
    echo "错误: 客户端编译失败"
    exit 1
fi
cd ..

# 编译示例程序
echo "编译示例程序..."
go build -o examples examples.go
if [ $? -ne 0 ]; then
    echo "错误: 示例程序编译失败"
    exit 1
fi

echo ""
echo "编译完成！"
echo ""

# 运行示例程序测试
echo "运行示例程序测试..."
./examples
if [ $? -ne 0 ]; then
    echo "错误: 示例程序运行失败"
    exit 1
fi

echo ""
echo "示例程序测试完成！"
echo ""

# 显示使用说明
echo "使用说明:"
echo "1. 启动服务端: ./server/rtsp-server"
echo "2. 运行客户端: ./client/rtsp-client"
echo "3. 使用VLC测试: vlc rtsp://localhost:8554/test"
echo ""

echo "测试完成！"
