# ── 构建阶段 ────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

# 安装必要工具
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# 先复制 go.mod，利用缓存层
COPY go.mod go.sum* ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=2.0.0" \
    -o moderation-server ./cmd/server

# ── 运行阶段（最小镜像）────────────────────────────────────
FROM scratch

# 时区 & HTTPS 证书
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 复制编译产物
COPY --from=builder /build/moderation-server /moderation-server

# 日志目录
VOLUME ["/logs"]

EXPOSE 8080

ENV TZ=Asia/Shanghai

ENTRYPOINT ["/moderation-server"]
