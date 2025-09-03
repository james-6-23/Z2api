# 使用官方 Deno 运行时作为基础镜像
FROM denoland/deno:1.45.5

# 设置工作目录
WORKDIR /app

# 设置环境变量
ENV PORT=8000

# 复制应用文件
COPY app.ts .
COPY deno.json .

# 预缓存依赖（可选，提高启动速度）
RUN deno cache app.ts

# 暴露端口
EXPOSE 8000

# 启动应用
CMD ["deno", "run", "--allow-net", "--allow-env", "app.ts"]
