# 使用官方 Go 运行时作为基础镜像
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY main.go .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 使用轻量级的 alpine 镜像作为运行时
FROM alpine:latest

# 安装 ca-certificates 和 curl（用于健康检查）
RUN apk --no-cache add ca-certificates curl

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 设置环境变量
ENV PORT=8000

# 暴露端口
EXPOSE 8000

# 启动应用
CMD ["./main"]
