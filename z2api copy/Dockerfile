# Z2API Docker 多阶段构建配置
# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖（移除不必要的包）
RUN apk add --no-cache ca-certificates tzdata

# 复制 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY main.go .

# 设置构建环境变量
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# 构建应用程序
RUN go build -a -installsuffix cgo -ldflags="-w -s" -o z2api main.go

# 验证构建结果（只使用 ls，不使用 file 命令）
RUN ls -la /app/z2api

# 第二阶段：运行时阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    curl \
    tzdata \
    && update-ca-certificates

# 创建非 root 用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/z2api .

# 设置文件权限
RUN chmod +x /app/z2api

# 切换到非 root 用户
USER appuser

# 设置环境变量
ENV TZ=Asia/Shanghai

# 暴露端口（应用硬编码为 8080）
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/v1/models || exit 1

# 启动应用
CMD ["./z2api"]
