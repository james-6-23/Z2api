# Z2API Go优化版 Dockerfile - 多阶段构建
FROM golang:1.21-alpine AS builder

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制源代码
COPY main.go .

# 初始化Go模块
RUN go mod init z2api-optimized
RUN go mod tidy

# 构建应用（静态链接）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o z2api-optimized main.go

# 运行阶段 - 使用最小镜像
FROM scratch

# 从builder阶段复制必要文件
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/z2api-optimized /z2api-optimized

# 设置环境变量默认值
ENV PORT=8080
ENV PERFORMANCE_MODE=balanced
ENV DEFAULT_KEY=123456
ENV MAX_CONCURRENT_CONNECTIONS=1000
ENV ANON_TOKEN_ENABLED=true
ENV THINK_TAGS_MODE=think
ENV ENABLE_DETAILED_LOGGING=true
ENV ENABLE_METRICS=true
ENV DEBUG_MODE=false
ENV STREAM_BUFFER_SIZE=16384
ENV MEMORY_LIMIT_MB=2048

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["/z2api-optimized", "-health-check"] || exit 1

# 启动命令
ENTRYPOINT ["/z2api-optimized"]
