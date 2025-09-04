# VGO-IAM Debug GUI Frontend

## 项目概述

这是VGO-IAM调试界面的前端项目，采用现代化的前后端分离架构设计。

## 技术栈

- **框架**: Vue.js 3 + TypeScript
- **构建工具**: Vite
- **UI组件库**: Element Plus
- **HTTP客户端**: Axios
- **路由**: Vue Router
- **状态管理**: Pinia
- **样式**: SCSS

## 项目结构

```
debug-gui-frontend/
├── public/                 # 静态资源
├── src/
│   ├── api/               # API接口定义
│   ├── components/        # 通用组件
│   ├── views/            # 页面组件
│   ├── router/           # 路由配置
│   ├── stores/           # 状态管理
│   ├── utils/            # 工具函数
│   ├── types/            # TypeScript类型定义
│   └── styles/           # 样式文件
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 功能模块

### 1. 用户管理
- 创建用户
- 查询用户
- 更新用户信息
- 删除用户
- 用户列表

### 2. 访问密钥管理
- 创建访问密钥
- 查看密钥列表
- 删除访问密钥
- 验证访问密钥

### 3. 权限管理
- 策略管理
- 权限检查
- 权限验证

### 4. 应用管理
- 创建应用
- 应用列表
- 更新应用
- 删除应用

### 5. 开发者认证
- 提交认证申请
- 查看认证状态

### 6. 系统监控
- 系统信息
- 健康检查
- 日志查看
- 指标监控

### 7. 配置管理
- 查看配置
- 更新配置

## API接口规范

所有API请求都需要包含签名认证信息：

```typescript
interface RequestConfig {
  headers: {
    'Authorization': string;     // 签名信息
    'X-IAM-Date': string;       // 时间戳
    'Content-Type': 'application/json';
  }
}
```

## 安全特性

1. **请求签名**: 所有API请求使用HMAC-SHA256签名
2. **时间窗口验证**: 请求时间戳验证，防止重放攻击
3. **HTTPS通信**: 生产环境强制使用HTTPS
4. **CORS配置**: 严格的跨域资源共享配置

## 开发指南

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

### 构建生产版本

```bash
npm run build
```

### 预览生产版本

```bash
npm run preview
```

## 部署说明

### 开发环境
- 后端API地址: `http://localhost:8081`
- 前端开发服务器: `http://localhost:3000`

### 生产环境
- 构建后的静态文件可部署到任何静态文件服务器
- 推荐使用Nginx进行反向代理和静态文件服务

## 环境配置

创建 `.env.local` 文件：

```env
# API基础地址
VITE_API_BASE_URL=http://localhost:8081

# 应用标题
VITE_APP_TITLE=VGO-IAM Debug GUI

# 是否启用开发模式
VITE_DEV_MODE=true
```

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。