#!/bin/bash

# 生成主密钥
MASTER_KEY=$(openssl rand -hex 32)
echo "Master Key: $MASTER_KEY"
echo "请将主密钥添加到配置文件 config/config.yaml"

# 生成AES密钥示例
AES_KEY=$(openssl rand -hex 16)
echo "AES Key: $AES_KEY"

# 生成访问密钥示例
ACCESS_KEY_ID=$(openssl rand -hex 10 | base64 | tr -d '=' | tr '+/' '_-' | head -c 20)
SECRET_ACCESS_KEY=$(openssl rand -hex 20 | base64 | tr -d '=' | tr '+/' '_-' | head -c 40)
echo "Access Key ID: $ACCESS_KEY_ID"
echo "Secret Access Key: $SECRET_ACCESS_KEY"