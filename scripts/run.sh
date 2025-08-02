#!/bin/bash

# 清理旧容器
docker-compose down -v

# 设置环境变量
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1

# 检查密钥
if [ -z "$MASTER_KEY" ]; then
  echo "⚠️ 使用不安全默认密钥，生产环境请设置 MASTER_KEY 环境变量"
  export MASTER_KEY="insecure_default_key_32bytes_xxxxxxxx"
fi

# 分阶段启动
echo "🚀 启动数据库服务..."
docker-compose up -d --build postgres redis

# 跨平台的等待函数
wait_for_postgres() {
  local timeout=60
  local start_time=$(date +%s)
  
  echo "⏳ 等待PostgreSQL准备就绪..."
  while ! docker-compose exec -T postgres pg_isready -U iam_user -d iam_db >/dev/null 2>&1; do
    local current_time=$(date +%s)
    local elapsed=$((current_time - start_time))
    
    if [ $elapsed -ge $timeout ]; then
      echo "❌ PostgreSQL启动超时"
      docker-compose logs postgres
      exit 1
    fi
    
    sleep 2
  done
}

# 调用等待函数
wait_for_postgres

echo "🔄 执行数据库迁移..."
docker-compose run --rm migrate

echo "🌐 启动应用服务..."
docker-compose up -d --build iam-service

echo "✅ 系统已就绪！容器状态："
docker-compose ps

# 获取服务访问信息
echo -e "\n访问信息："
echo "gRPC端点: localhost:50051"
echo "Redis内部地址: redis:6379"
echo "PostgreSQL内部地址: postgres:5432"
echo "PostgreSQL外部访问: docker-compose exec postgres psql -U iam_user -d iam_db"