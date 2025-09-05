# vgo-iam 架构设计文档

## 概述

vgo-iam 是一个基于 Go 语言开发的身份认证和访问管理系统，采用分层架构设计，支持 gRPC Gateway 和 RESTful API。

## 架构层次

### 1. 应用层 (App Layer)
- **位置**: `internal/app/`
- **职责**: 应用程序入口点，负责初始化各个组件
- **核心文件**: `app.go`

### 2. 路由层 (Router Layer)
- **位置**: `internal/router/`
- **职责**: HTTP 路由管理，中间件配置
- **核心文件**: 
  - `router.go` - 路由配置和管理
  - `manager.go` - 中间件和控制器管理器

### 3. 控制器层 (Controller Layer)
- **位置**: `internal/controller/`
- **职责**: 处理 HTTP 请求，参数验证，响应格式化
- **核心文件**:
  - `user_controller.go` - 用户管理
  - `policy_controller.go` - 策略管理
  - `access_key_controller.go` - 访问密钥管理
  - `application_controller.go` - 应用管理

### 4. 服务层 (Service Layer)
- **位置**: `internal/service/`
- **职责**: 业务逻辑处理，事务管理
- **核心文件**:
  - `user_service.go` - 用户业务逻辑
  - `policy_service.go` - 策略业务逻辑
  - `access_key_service.go` - 访问密钥业务逻辑
  - `application_service.go` - 应用业务逻辑

### 5. 存储层 (Store Layer)
- **位置**: `internal/store/`
- **职责**: 数据持久化，数据库操作
- **核心文件**:
  - `user_store.go` - 用户数据操作
  - `policy_store.go` - 策略数据操作
  - `access_key_store.go` - 访问密钥数据操作
  - `application_store.go` - 应用数据操作

### 6. 模型层 (Model Layer)
- **位置**: `internal/model/`
- **职责**: 数据模型定义，常量定义
- **核心文件**:
  - `user.go` - 用户模型
  - `policy.go` - 策略模型
  - `accesskey.go` - 访问密钥模型
  - `application.go` - 应用模型

## 核心组件

### 依赖注入容器
- **位置**: `internal/container/`
- **职责**: 管理服务依赖关系，提供统一的服务获取接口

### 中间件系统
- **位置**: `internal/middleware/`
- **功能**:
  - 身份认证 (`auth.go`)
  - 跨域处理 (`cors.go`)
  - 错误处理 (`error_handler.go`)
  - 日志记录 (`logger.go`)
  - 限流控制 (`rate_limiter.go`)
  - 安全防护 (`security.go`)

### 策略引擎
- **位置**: `internal/policy/`
- **职责**: 权限策略评估和执行

### 配置管理
- **位置**: `internal/config/`
- **职责**: 应用配置加载和管理

## 数据库设计

### 数据库驱动
- 使用 `github.com/gocraft/dbr/v2` 作为数据库驱动
- 支持 PostgreSQL 数据库

### 核心表结构
1. **users** - 用户表
2. **policies** - 策略表
3. **access_keys** - 访问密钥表
4. **applications** - 应用表
5. **developer_verifications** - 开发者验证表
6. **temporary_credentials** - 临时凭证表

## API 设计

### gRPC Gateway
- 使用 buf 管理 proto 文件
- 自动生成 RESTful API 接口
- 支持 HTTP/1.1 和 gRPC 双协议

### 路由分组
1. **用户管理** (`/api/v1/users`)
   - 用户注册、登录、信息管理
   - 密码重置、邮箱验证

2. **策略管理** (`/api/v1/policies`)
   - 策略创建、更新、删除
   - 策略版本管理

3. **访问密钥管理** (`/api/v1/access-keys`)
   - 密钥创建、删除、查询
   - 使用统计和元数据管理

4. **应用管理** (`/api/v1/applications`)
   - 应用注册、更新、删除
   - 凭证管理和状态控制

5. **STS服务** (`/api/v1/sts`)
   - 临时凭证颁发
   - 令牌刷新和撤销

## 安全特性

### 身份认证
- JWT Token 认证
- 多因素认证支持
- 会话管理

### 权限控制
- 基于策略的访问控制 (PBAC)
- 细粒度权限管理
- 角色继承机制

### 安全防护
- CORS 跨域保护
- 请求限流
- SQL 注入防护
- XSS 攻击防护

## 部署架构

### 容器化部署
- Docker 容器化
- Docker Compose 本地开发
- Kubernetes 生产部署

### 环境配置
- 开发环境: macOS
- 生产环境: Linux + Kubernetes

## 开发规范

### 代码规范
- 函数级注释，包含功能、参数、返回值说明
- 简洁易读的代码风格
- 统一的错误处理机制

### 文档管理
- 使用 VuePress 构建文档
- 需求变更时及时更新文档

### 构建工具
- 使用 buf 构建和管理 proto 文件
- 自动化 CI/CD 流程

## 性能优化

### 数据库优化
- 连接池管理
- 查询优化
- 索引设计

### 缓存策略
- Redis 缓存
- 本地缓存
- 缓存失效策略

### 监控告警
- 性能指标监控
- 错误日志收集
- 健康检查机制

## 扩展性设计

### 微服务架构
- 服务拆分策略
- 服务间通信
- 配置中心

### 插件机制
- 中间件插件
- 存储插件
- 认证插件

---

*本文档会随着项目发展持续更新*