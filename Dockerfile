# 多阶段构建
FROM golang:1.24.1-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制go.work文件和模块
COPY go.work go.work.sum ./
COPY vgo-kit/ ./vgo-kit/
COPY vgo-iam/ ./vgo-iam/

# 设置工作目录到vgo-iam
WORKDIR /app/vgo-iam

# 下载依赖
RUN go mod download

# 构建应用
# 使用构建参数注入版本信息
ARG VERSION=v1.0.0
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -ldflags "-X github.com/vera-byte/vgo-iam/internal/version.Version=${VERSION} \
    -X github.com/vera-byte/vgo-iam/internal/version.Commit=${COMMIT} \
    -X github.com/vera-byte/vgo-iam/internal/version.BuildTime=${BUILD_TIME}" \
    -o vgo-iam ./cmd

# 运行阶段
FROM alpine:latest

# 安装必要的包
RUN apk --no-cache add ca-certificates tzdata curl

# 安装grpc_health_probe
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.24 && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi && \
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-${ARCH} && \
    chmod +x /bin/grpc_health_probe

# 安装golang-migrate工具
RUN MIGRATE_VERSION=v4.17.0 && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi && \
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi && \
    wget -qO- https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${ARCH}.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

# 创建非root用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/vgo-iam/vgo-iam .
COPY --from=builder /app/vgo-iam/config ./config
COPY --from=builder /app/vgo-iam/migrations ./migrations

# 设置二进制文件执行权限
RUN chmod +x ./vgo-iam

# 创建日志目录
RUN mkdir -p logs && chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 50051 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD grpc_health_probe -addr=localhost:50051 || exit 1

# 启动应用
CMD ["./vgo-iam", "server"]