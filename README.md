iam-system/
├── buf.gen.yaml
├── buf.yaml
├── cmd
│   └── server
│       └── main.go
├── config
│   └── config.yaml
├── internal
│   ├── api
│   │   └── rpc_service.go
│   ├── auth
│   │   ├── middleware.go
│   │   └── signature_v4.go
│   ├── crypto
│   │   ├── key_crypto.go
│   │   └── key_rotation.go
│   ├── model
│   │   ├── accesskey.go
│   │   ├── policy.go
│   │   └── user.go
│   ├── policy
│   │   └── engine.go
│   ├── service
│   │   ├── accesskey.go
│   │   ├── policy.go
│   │   └── user.go
│   ├── store
│   │   ├── accesskey_store.go
│   │   ├── policy_store.go
│   │   ├── postgres.go
│   │   └── user_store.go
│   └── util
│       └── util.go
├── migrations
│   ├── 000001_init_schema.down.sql
│   └── 000001_init_schema.up.sql
├── proto
│   └── iam.proto
└── scripts
    └── generate_keys.sh




SQL迁移工具使用说明
# 使用 migrate 工具执行迁移
1. 安装 migrate 工具：

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```
2. 执行迁移：

```bash
migrate -path /Users/mac/workspace/vgo-iam/migrations/ -database "postgres://postgres:maile@321@10.0.0.200:5432/vgo_iam?sslmode=disable" up
```
3. 回滚迁移：

```bash
migrate -path migrations -database "postgres://username:password@localhost:5432/iam_db?sslmode=disable" down 1
```
3. 使用 Docker 执行迁移
```bash
docker run -v $(pwd)/migrations:/migrations --network host migrate/migrate \
  -path=/migrations/ \
  -database "postgres://username:password@localhost:5432/iam_db?sslmode=disable" up
```

4.
```bash
go build -ldflags "-X github.com/vera-byte/vgo-iam/internal/version.Version=1.2.3 -X github.com/vera-byte/vgo-iam/internal/version.Commit=$(git rev-parse HEAD) -X github.com/vera-byte/vgo-iam/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

```

# 手动备份数据库
```bash
docker-compose exec postgres pg_dump -U iam_user -d iam_db > backup.sql
```
关键设计说明
数据完整性保障：

使用外键约束确保关联数据完整性

添加 ON DELETE CASCADE 自动清理关联数据

使用 CHECK 约束限制状态字段值

性能优化：

为所有查询条件创建索引

为状态字段和关联ID字段创建索引

审计字段：

包含 created_at 和 updated_at 时间戳

预留 last_used_at 用于后续审计

安全考虑：

密钥字段使用 encrypted_ 前缀明确标识加密数据

密码字段预留但不强制使用

扩展性设计：

策略文档使用 JSONB 类型存储，支持灵活的策略定义

表结构设计支持后续添加审计日志等功能

这套迁移文件提供了完整的 IAM 系统数据存储方案，支持用户管理、策略管理和访问密钥管理等核心功能，同时考虑了性能、安全性和可扩展性需求。