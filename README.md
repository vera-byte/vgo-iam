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

安全密钥生成
```bash
./scripts/generate_keys.sh
```


SQL迁移工具使用说明
# 使用 migrate 工具执行迁移
1. 安装 migrate 工具：

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```
2. 执行迁移：

```bash
migrate -path /Users/mac/workspace/vgo_micro_service/vgo-iam/migrations/ -database "postgres://vgo_iam:KESdCZeYYXBZcebH@10.0.0.200:5432/vgo_iam?sslmode=disable" up
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

**事务支持**：init admin 命令现已支持数据库事务，确保管理员用户初始化过程中的数据一致性。如果在创建用户、设置密码、创建策略或生成访问密钥的任何步骤中发生错误，系统将自动回滚所有已执行的操作，保证数据库状态的完整性

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

## 加签和验签过程

### 整体架构概述

VGO-IAM 系统实现了一套基于 HMAC-SHA256 的 V4 签名算法，类似于 AWS IAM 的访问控制机制。该系统通过 gRPC 提供服务，使用 metadata 传递认证信息。

### 加签过程

加签过程主要在客户端完成，核心步骤如下：

#### 1. 准备请求数据

首先，客户端需要准备要发送的请求数据，并将其序列化为 JSON 字符串。

```go
// 示例：将请求对象序列化为JSON
reqData, err := json.Marshal(req)
if err != nil {
    log.Fatalf("序列化请求数据失败: %v", err)
}
```

#### 2. 生成签名

签名生成通过 `signature.SignV4` 函数完成，该函数实现了以下步骤：

1. **获取当前时间戳**：使用当前时间的 Unix 时间戳

2. **构建待签名字符串**：格式为 `IAM-HMAC-SHA256\n{timestamp}\n{requestDataHash}`，其中 `requestDataHash` 是请求数据的 SHA256 哈希值

3. **派生签名密钥**：使用多重 HMAC-SHA256 运算派生密钥：
   - 首先，使用日期和基础密钥生成日期密钥
   - 然后，使用日期密钥和区域（default）生成区域密钥
   - 接着，使用区域密钥和服务名称（iam）生成服务密钥
   - 最后，使用服务密钥和请求类型（request）生成最终签名密钥

4. **计算最终签名**：使用派生的签名密钥对"待签名字符串"进行 HMAC-SHA256 运算，并将结果进行 Base64 编码

```go
// 签名核心代码
func SignV4(accessKeyID, secretKey, requestData string) SignV4Result {
    timestamp := time.Now().Unix()
    
    // 构建待签字符串
    stringToSign := buildStringToSign(timestamp, requestData)
    
    // 计算签名
    return SignV4Result{
        SecretKey:   secretKey,
        AccessKeyID: accessKeyID,
        Signature:   calculateSignature(stringToSign, secretKey, timestamp),
        Timestamp:   timestamp,
    }
}
```

#### 3. 添加认证信息到上下文

最后，将生成的签名信息添加到 gRPC 调用的 metadata 中，以便服务端验证：

```go
// 将签名信息添加到metadata
md := metadata.Pairs(
    "access-key-id", signer.AccessKeyID,
    "signature", signer.Signature,
    "x-iam-date", strconv.FormatInt(signer.Timestamp, 10),
    "request-data", string(reqData),
)
ctx = metadata.NewOutgoingContext(ctx, md)
```

### 验签过程

验签过程在服务端进行，主要通过 `AccessKeyInterceptor` 拦截器和 `VerifySignV4` 函数完成：

#### 1. 提取认证信息

服务端从 gRPC 调用的 metadata 中提取访问密钥 ID、签名、时间戳和请求数据：

```go
accessKeyID := vgokit.GetMetadataValue(ctx, "access-key-id")
sign := vgokit.GetMetadataValue(ctx, "signature")
timestamp := vgokit.GetMetadataValue(ctx, "x-iam-date")
requestData := vgokit.GetMetadataValue(ctx, "request-data")
```

#### 2. 验证时间戳

首先验证时间戳是否在允许的时间窗口内（默认 ±5 分钟），防止重放攻击：

```go
func verifyTimestamp(timestamp int64) bool {
    // 转换为时间
    t := time.Unix(timestamp, 0)
    
    // 检查时间戳是否在允许范围内（±5分钟）
    now := time.Now()
    diff := now.Sub(t)
    return diff.Abs() <= 5*time.Minute
}
```

#### 3. 验证访问密钥

根据访问密钥 ID 从存储中获取对应的密钥信息，并检查密钥状态是否为 "active"：

```go
// 获取访问密钥
ak, err := akStore.GetByAccessKeyID(accessKeyID)
if err != nil {
    return nil, status.Error(codes.Unauthenticated, "非法访问,请检查访问密钥是否正确")
}

// 验证密钥状态
if ak.Status != "active" {
    return nil, status.Error(codes.PermissionDenied, "非法访问,请检查密钥状态是否正确")
}
```

#### 4. 验证签名

最后，使用 `VerifySignV4` 函数验证签名是否有效：

```go
// 验证签名
valid, err := signature.VerifySignV4(sign, requestData, timestamp, string(ak.SecretAccessKey))
if err != nil || !valid {
    return nil, status.Error(codes.Unauthenticated, "签名验证失败,请检查签名后再试!")
}
```

`VerifySignV4` 函数的验证逻辑与加签逻辑对称：
1. 解析时间戳
2. 验证时间窗口
3. 重新构建待签名字符串
4. 使用相同的算法计算签名
5. 比较计算得到的签名与请求中的签名是否一致

### 安全特性

该签名机制具有以下安全特性：

1. **防重放攻击**：通过时间戳验证，确保请求只能在有限时间内有效
2. **密钥派生机制**：使用多重 HMAC-SHA256 派生密钥，增强安全性
3. **完整性验证**：通过对请求数据进行哈希和签名，确保数据在传输过程中未被篡改
4. **密钥状态检查**：验证访问密钥是否处于活动状态

## 数据库权限问题处理

如果遇到如下错误：

```
pq: permission denied for schema public
```

可以通过以下 SQL 命令解决：

```sql
-- 连接到 vgo_iam 数据库
\c vgo_iam

-- 授权 schema 使用权限
GRANT USAGE ON SCHEMA public TO vgo_iam;

-- 授权在 schema 内创建表的权限
GRANT CREATE ON SCHEMA public TO vgo_iam;
```