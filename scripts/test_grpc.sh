#!/bin/bash

# 确保服务正在运行
echo "检查gRPC服务是否在运行..."
if ! lsof -i :50051 > /dev/null; then
  echo "错误: gRPC服务未在端口50051上运行。请先启动服务。"
  exit 1
fi

echo "gRPC服务正在运行，开始测试..."

echo -e "\n测试连接到服务..."
grpcurl -plaintext -v localhost:50051 list

# 测试创建用户
 echo -e "\n1. 测试创建用户..."
grpcurl -plaintext -v -d '{"name":"testuser","display_name":"Test User","email":"test@example.com"}' localhost:50051 iam.v1.IAM/CreateUser

# 测试获取用户
 echo -e "\n2. 测试获取用户..."
grpcurl -plaintext -v -d '{"name":"testuser"}' localhost:50051 iam.v1.IAM/GetUser

# 测试创建策略
 echo -e "\n3. 测试创建策略..."
grpcurl -plaintext -v -d '{"name":"testpolicy","description":"Test Policy","policy_document":"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"iam:*\"],\"Resource\":[\"*\"]}]}"}' localhost:50051 iam.v1.IAM/CreatePolicy

# 测试附加策略到用户
 echo -e "\n4. 测试附加策略到用户..."
grpcurl -plaintext -v -d '{"user_name":"testuser","policy_name":"testpolicy"}' localhost:50051 iam.v1.IAM/AttachUserPolicy

# 测试创建访问密钥
 echo -e "\n5. 测试创建访问密钥..."
grpcurl -plaintext -v -d '{"user_name":"testuser"}' localhost:50051 iam.v1.IAM/CreateAccessKey

# 测试列出访问密钥
 echo -e "\n6. 测试列出访问密钥..."
grpcurl -plaintext -v -d '{"user_name":"testuser"}' localhost:50051 iam.v1.IAM/ListAccessKeys

# 测试权限检查
 echo -e "\n7. 测试权限检查..."
grpcurl -plaintext -v -d '{"user_name":"testuser","action":"iam:CreateUser","resource":"*"}' localhost:50051 iam.v1.IAM/CheckPermission

echo -e "\n测试完成。"