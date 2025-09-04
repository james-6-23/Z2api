# Z2API Deno优化版 Dockerfile
FROM denoland/deno:1.40.2

# 设置工作目录
WORKDIR /app

# 复制应用文件
COPY app.ts .

# 设置环境变量默认值
ENV PORT=8080
ENV PERFORMANCE_MODE=balanced
ENV DEFAULT_KEY=123456
ENV ANON_TOKEN_ENABLED=true
ENV THINK_TAGS_MODE=think
ENV ENABLE_DETAILED_LOGGING=true
ENV DEBUG_MODE=false

# 缓存依赖（提高构建速度）
RUN deno cache app.ts

# 使用已存在的deno用户（Deno官方镜像已包含）
USER deno

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD deno eval "fetch('http://localhost:8080/health').then(r => r.ok ? Deno.exit(0) : Deno.exit(1)).catch(() => Deno.exit(1))"

# 启动命令
CMD ["run", "--allow-net", "--allow-env", "app.ts"]
