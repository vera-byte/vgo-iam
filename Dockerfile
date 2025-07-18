# 使用多阶段构建
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

# 安装依赖和构建
RUN apk add --no-cache git make \
    && go mod download \
    && CGO_ENABLED=0 GOOS=linux go build -o /iam-service ./cmd/server

# 最终镜像
FROM alpine:3.16

WORKDIR /app
COPY --from=builder /iam-service /app/iam-service
COPY --from=builder /app/config /app/config

# 安装 grpc_health_probe
RUN apk add --no-cache curl \
    && curl -L https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.14/grpc_health_probe-linux-amd64 -o /bin/grpc_health_probe \
    && chmod +x /bin/grpc_health_probe \
    && apk del curl

EXPOSE 50051
CMD ["/app/iam-service"]