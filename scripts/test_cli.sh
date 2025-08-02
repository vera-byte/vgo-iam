#!/bin/bash

# 构建项目
echo "Building project..."
go build -o vgo-iam main.go

# 测试命令行功能
echo -e "\nTesting command line functionality..."

# 测试创建用户命令
echo "\n1. Creating a test user..."
./vgo-iam server --create-user testuser --no-server

# 测试获取用户命令
echo "\n2. Getting user information..."
./vgo-iam server --get-user testuser --no-server

# 测试获取用户策略命令
echo "\n3. Getting user policies..."
./vgo-iam server --get-policies testuser --no-server

# 测试启动服务器
echo "\n4. Starting server (press Ctrl+C to stop)..."
./vgo-iam server

# 清理
echo "\nCleaning up..."
rm -f vgo-iam

echo "\nTest completed."