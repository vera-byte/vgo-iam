# VGO 微服务项目

一个基于 Go 语言开发的微服务架构项目，提供身份认证与访问管理（IAM）服务和通用工具包。

## 项目概述

本项目采用微服务架构设计，包含以下核心组件：

- **vgo-iam**: 身份认证与访问管理服务，提供用户管理、策略管理、访问密钥管理和权限验证功能
- **vgo-kit**: 通用工具包，提供配置管理、数据库连接、日志记录、错误监控等基础功能

## 项目架构

```
vgo_micro_service/
├── vgo-iam/           # IAM 微服务
│   ├── cmd/           # 命令行入口
│   ├── internal/      # 内部业务逻辑
│   │   ├── api/       # API 层
│   │   ├── auth/      # 认证中间件
│   │   ├── service/   # 业务服务层
│   │   ├── store/     # 数据存储层
│   │   └── model/     # 数据模型
│   ├── proto/         # gRPC 协议定义
│   ├── migrations/    # 数据库迁移文件
│   ├── config/        # 配置文件
│   └── scripts/       # 部署和测试脚本
└── vgo-kit/           # 通用工具包
    ├── config/        # 配置管理
    ├── db/            # 数据库工具
    └── sentry/        # 日志和监控
```

## 核心功能

### VGO-IAM 服务

- **用户管理**: 创建、查询用户信息
- **策略管理**: 创建策略、关联用户策略
- **访问密钥管理**: 创建、列出、更新访问密钥状态
- **应用管理**: 创建、查询、更新、删除OAuth2应用
- **权限验证**: 验证访问密钥、检查用户权限
- **gRPC API**: 提供高性能的 gRPC 接口
- **认证中间件**: 基于访问密钥的请求认证
- **调试界面**: 提供Web界面进行管理操作

### VGO-Kit 工具包

- **配置管理**: 基于 Viper 的配置加载
- **数据库连接**: PostgreSQL 连接管理
- **日志系统**: 基于 Zap 的结构化日志
- **错误监控**: 集成 Sentry 错误追踪
- **OpenTelemetry**: 分布式追踪支持

## 技术栈

- **语言**: Go 1.24.1
- **框架**: gRPC, Cobra CLI
- **数据库**: PostgreSQL
- **缓存**: Redis
- **日志**: Zap + Lumberjack
- **监控**: Sentry + OpenTelemetry
- **配置**: Viper
- **容器化**: Docker + Docker Compose
- **数据库迁移**: golang-migrate

## 快速开始

### 环境要求

- Go 1.24.1+
- PostgreSQL 13+
- Redis 6+
- Docker & Docker Compose (可选)

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd vgo_micro_service
```

2. **启动依赖服务**
```bash
cd vgo-iam
docker-compose up -d postgres redis
```

3. **运行数据库迁移**
```bash
docker-compose up migrate
```

4. **配置服务**
```bash
cp config/config.yaml.example config/config.yaml
# 编辑配置文件，设置数据库连接等信息
```

5. **构建和运行服务**
```bash
# 构建
go build -o bin/iam-service cmd/main.go

# 运行服务
./bin/iam-service server
```

### 使用 Docker 部署

```bash
cd vgo-iam
docker-compose up -d
```

## 配置说明

### 数据库配置
```yaml
database:
  dsn: "host=localhost port=5432 user=postgres password=postgres dbname=vgo_iam sslmode=disable"
```

### gRPC 服务配置
```yaml
grpc:
  port: "50051"
```

### 安全配置
```yaml
security:
  master_key: "your-master-key-here"  # 用于加密访问密钥
```

### 日志配置
```yaml
log:
  level: "info"
  directory: "./logs"
  filename: "app.log"
  to_stdout: true
```

## API 使用示例

### 创建用户
```bash
grpcurl -plaintext -d '{
  "name":"testuser",
  "display_name":"Test User",
  "email":"test@example.com"
}' localhost:50051 iam.v1.IAM/CreateUser
```

### 创建策略
```bash
grpcurl -plaintext -d '{
  "name":"testpolicy",
  "description":"Test Policy",
  "policy_document":"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"iam:*\"],\"Resource\":[\"*\"]}]}"
}' localhost:50051 iam.v1.IAM/CreatePolicy
```

### 创建访问密钥
```bash
grpcurl -plaintext -d '{
  "user_name":"testuser"
}' localhost:50051 iam.v1.IAM/CreateAccessKey
```

### 检查权限
```bash
grpcurl -plaintext -d '{
  "user_name":"testuser",
  "action":"iam:CreateUser",
  "resource":"*"
}' localhost:50051 iam.v1.IAM/CheckPermission
```

### 应用管理

#### 创建应用
```bash
grpcurl -plaintext -d '{
  "app_name":"MyTestApp",
  "app_description":"我的测试应用",
  "app_type":"web",
  "app_website":"https://example.com",
  "callback_urls":["https://example.com/callback"],
  "allowed_origins":["https://example.com"]
}' localhost:50051 iam.v1.IAM/CreateApplication
```

#### 查询应用
```bash
grpcurl -plaintext -d '{
  "app_id":1
}' localhost:50051 iam.v1.IAM/GetApplication
```

#### 获取应用列表
```bash
grpcurl -plaintext -d '{
  "user_name":"testuser",
  "status":"active",
  "page":1,
  "page_size":10
}' localhost:50051 iam.v1.IAM/ListApplications
```

#### 更新应用
```bash
grpcurl -plaintext -d '{
  "app_id":1,
  "app_name":"UpdatedApp",
  "app_description":"更新后的应用描述",
  "app_type":"spa"
}' localhost:50051 iam.v1.IAM/UpdateApplication
```

#### 删除应用
```bash
grpcurl -plaintext -d '{
  "app_id":1
}' localhost:50051 iam.v1.IAM/DeleteApplication
```

### 使用调试界面
```bash
# 启动调试GUI
./bin/iam-service debug-gui

# 访问Web界面: http://localhost:8080
# 提供用户管理、访问密钥管理、权限检查和应用管理的Web界面
```

## 数据库结构

### 用户表 (users)
- id: 主键
- name: 用户名（唯一）
- display_name: 显示名称
- email: 邮箱（唯一）
- created_at/updated_at: 时间戳

### 策略表 (policies)
- id: 主键
- name: 策略名称（唯一）
- description: 策略描述
- policy_document: 策略文档（JSON）
- created_at/updated_at: 时间戳

### 访问密钥表 (access_keys)
- id: 主键
- user_id: 用户ID（外键）
- access_key_id: 访问密钥ID
- encrypted_secret_access_key: 加密的密钥
- status: 状态（active/inactive）
- created_at/updated_at: 时间戳

### 用户策略关联表 (user_policies)
- user_id: 用户ID（外键）
- policy_id: 策略ID（外键）

### 应用表 (applications)
- id: 主键
- user_id: 用户ID（外键）
- app_name: 应用名称
- app_description: 应用描述
- app_type: 应用类型（web/mobile/api/spa）
- app_icon_url: 应用图标URL
- app_website: 应用网站URL
- status: 应用状态（active/inactive/suspended）
- callback_urls: 回调URL列表（JSON数组）
- allowed_origins: 允许的来源列表（JSON数组）
- created_at/updated_at: 时间戳

## 开发指南

### 项目结构说明

- `cmd/`: 命令行入口点
- `internal/`: 内部业务逻辑，不对外暴露
- `pkg/`: 可对外暴露的包
- `proto/`: gRPC 协议定义文件
- `migrations/`: 数据库迁移文件
- `scripts/`: 部署和测试脚本

### 添加新的 API

1. 在 `proto/iam.proto` 中定义新的 RPC 方法
2. 运行 `buf generate` 生成代码
3. 在 `internal/service/` 中实现业务逻辑
4. 在 `internal/store/` 中实现数据访问
5. 添加相应的测试

### 运行测试

```bash
# 运行单元测试
go test ./...

# 运行 gRPC 集成测试
./scripts/test_grpc.sh
```

## 部署和运维

### 生产环境部署

1. **环境变量配置**
```bash
export DB_HOST=your-db-host
export DB_PORT=5432
export DB_USER=your-db-user
export DB_PASSWORD=your-db-password
export DB_NAME=vgo_iam
export MASTER_KEY=your-secure-master-key
```

2. **使用 Docker 部署**
```bash
docker build -t vgo-iam .
docker run -d -p 50051:50051 \
  -e DB_HOST=your-db-host \
  -e DB_USER=your-db-user \
  -e DB_PASSWORD=your-db-password \
  vgo-iam
```

### 监控和日志

- 日志文件位置: `logs/iam.log`
- Sentry 错误监控: 配置 DSN 后自动上报错误
- 健康检查: gRPC 健康检查端点

### 初始化管理员

```bash
# 运行初始化脚本创建管理员用户
./scripts/init_admin.sh
```

## 安全考虑

- 访问密钥使用主密钥加密存储
- gRPC 请求需要有效的访问密钥认证
- 策略文档支持细粒度权限控制
- 支持访问密钥的启用/禁用状态管理
- **数据库事务支持**: init admin 命令现已支持数据库事务，确保管理员初始化过程的原子性操作，任何步骤失败都会自动回滚

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系方式

- 项目维护者: [vera-byte](https://github.com/vera-byte)
- 问题反馈: 请使用 GitHub Issues

## 更新日志

### v1.2.0
- **新增功能**: 应用管理模块
  - 支持创建、查询、更新、删除OAuth2应用
  - 提供完整的应用生命周期管理
  - 支持多种应用类型（web、mobile、api、spa）
  - 支持回调URL和CORS配置管理
- **Web界面增强**: 调试GUI新增应用管理界面
  - 提供友好的Web界面进行应用管理操作
  - 完整的输入验证和错误处理
  - 支持分页查询和状态筛选
- **API扩展**: 新增5个应用管理相关的gRPC接口
- **数据库**: 新增applications表支持应用数据存储

### v1.1.0
- **新增功能**: init admin 命令支持数据库事务
- **改进**: 管理员初始化过程现在具备原子性，确保数据一致性
- **安全性**: 增强了错误处理和回滚机制

### v1.0.0
- 初始版本发布
- 实现基础的 IAM 功能
- 支持用户、策略、访问密钥管理
- 提供 gRPC API 接口
- 集成日志和监控系统