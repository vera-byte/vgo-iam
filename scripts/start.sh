#!/bin/bash

# 打包并启动VGO-IAM服务的脚本

set -e  # 遇到错误立即退出

# 设置字符编码
export LANG=en_US.UTF-8

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 检查是否安装了Go
if ! command -v go &> /dev/null
then
    echo -e "${RED}错误: 未找到Go编译器。请先安装Go。${NC}"
    exit 1
fi

# 检查Go版本
GO_VERSION=$(go version | cut -d ' ' -f 3 | sed 's/go//')
REQUIRED_VERSION="1.24"
if [[ "$GO_VERSION" < "$REQUIRED_VERSION" ]]; then
    echo -e "${RED}错误: Go版本过低。需要至少${REQUIRED_VERSION}，当前版本是${GO_VERSION}。${NC}"
    exit 1
fi

# 打印当前目录
echo -e "${BLUE}当前工作目录: $(pwd)${NC}"

# 创建bin目录（如果不存在）
mkdir -p ./bin

# 获取版本信息
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date +"%Y-%m-%dT%H:%M:%SZ")
# 直接设置版本号，因为从文件中提取可能有问题
VERSION="v1.0.0"

# 编译项目，注入版本信息
 echo -e "${YELLOW}正在编译项目...${NC}"
 echo -e "${BLUE}版本: ${VERSION}, 提交: ${COMMIT}, 构建时间: ${BUILD_TIME}${NC}"
CGO_ENABLED=0 go build -ldflags "-X 'github.com/vera-byte/vgo-iam/internal/version.Commit=${COMMIT}' -X 'github.com/vera-byte/vgo-iam/internal/version.BuildTime=${BUILD_TIME}'" -o ./bin/iam-service ./cmd/server/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败!${NC}"
    exit 1
fi

# 确保配置文件存在
if [ ! -f ./config/config.yaml ]; then
    echo -e "${YELLOW}警告: 配置文件config/config.yaml不存在。正在创建默认配置...${NC}"
    mkdir -p ./config
    cat > ./config/config.yaml << EOF
server:
  port: 50051

logger:
  level: info
  output_path: ./logs/iam.log
  max_size: 100
  max_age: 7
  max_backups: 3

postgres:
  host: localhost
  port: 5433
  user: iam_user
  password: iam_password
  dbname: iam_db
  sslmode: disable

redis:
  addr: localhost:6379
  db: 0
EOF
    echo -e "${GREEN}已创建默认配置文件。${NC}"
fi

# 确保日志目录存在
mkdir -p ./logs

# 检查端口是否被占用
PORT=50051
if lsof -i :${PORT} > /dev/null; then
    echo -e "${YELLOW}警告: 端口 ${PORT} 已被占用。${NC}"
    echo -e "${YELLOW}占用端口 ${PORT} 的进程:${NC}"
    lsof -i :${PORT} | grep LISTEN
    
    # 自动关闭占用端口的进程
    echo -e "${GREEN}正在自动关闭占用端口 ${PORT} 的进程...${NC}"
    PID=$(lsof -t -i :${PORT})
    if [ -n "${PID}" ]; then
        kill -9 ${PID}
        echo -e "${GREEN}进程 ${PID} 已关闭。${NC}"
        # 等待片刻，确保进程已完全关闭
        sleep 2
    else
        echo -e "${RED}无法获取占用端口的进程ID。${NC}"
        exit 1
    fi
fi

# 启动服务（协程方式）
echo -e "${GREEN}编译成功! 正在后台启动服务...${NC}"
echo -e "${BLUE}服务将在端口 ${PORT} 上运行。${NC}"
echo -e "${BLUE}日志将写入: ./logs/iam.log${NC}"
echo -e "${BLUE}使用'ps aux | grep iam-service'查看服务进程。${NC}"
echo -e "${BLUE}使用'kill -9 <PID>'停止服务。${NC}"

# 后台启动服务
./bin/iam-service > ./logs/iam.log 2>&1 &

# 记录服务PID
SERVICE_PID=$!
echo -e "${GREEN}服务已在后台启动，PID: ${SERVICE_PID}${NC}"

# 检查服务是否成功启动
sleep 2
if ps -p $SERVICE_PID > /dev/null; then
    echo -e "${GREEN}服务启动成功!${NC}"
else
    echo -e "${RED}服务启动失败! 请查看日志获取详细信息: ./logs/iam.log${NC}"
    exit 1
fi