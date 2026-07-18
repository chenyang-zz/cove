# syntax=docker/dockerfile:1
# =============================================================================
# Cove API - Multi-stage Dockerfile
# =============================================================================
# Build stage: 编译 Go 二进制
#   CMD_TARGET: api | worker | gateway（通过 --build-arg 切换入口）
# =============================================================================
FROM golang:1.25-alpine AS builder

ARG CMD_TARGET=api
ARG CMD_TARGET

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 先复制依赖文件，利用 Docker layer cache 加速
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制源码
COPY . .

# 静态编译，关闭 CGO，减小体积；输出统一为 app，由 build arg 决定构建目标
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /usr/local/bin/app \
    ./cmd/${CMD_TARGET}

# =============================================================================
# Runtime stage: 最小化运行时镜像
# =============================================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget curl \
    && addgroup -S cove && adduser -S cove -G cove

WORKDIR /app

COPY --from=builder /usr/local/bin/app /usr/local/bin/app
COPY --from=builder /app/configs/config.yml.example /app/configs/config.yml.example

USER cove

# api 和 gateway 服务端口；compose 按服务决定是否 publish
EXPOSE 8000 8010

# 默认 healthcheck 用于 api，worker/gateway 服务在 compose 中覆盖
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8000/api/health || exit 1

ENTRYPOINT ["app"]
