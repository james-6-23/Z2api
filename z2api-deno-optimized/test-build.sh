#!/bin/bash

# Z2API Deno优化版 - 构建测试脚本

echo "🚀 开始构建 Z2API Deno 优化版..."

# 清理旧的容器和镜像
echo "🧹 清理旧的容器和镜像..."
docker compose down --remove-orphans
docker rmi z2api-deno-optimized-z2api-deno 2>/dev/null || true

# 构建新镜像
echo "🔨 构建新镜像..."
docker compose build --no-cache

if [ $? -eq 0 ]; then
    echo "✅ 镜像构建成功！"
    
    # 启动服务
    echo "🚀 启动服务..."
    docker compose up -d
    
    if [ $? -eq 0 ]; then
        echo "✅ 服务启动成功！"
        echo "📊 检查服务状态..."
        sleep 5
        docker compose ps
        
        echo "🏥 测试健康检查..."
        curl -s http://localhost:8080/health | jq . || echo "健康检查响应获取中..."
        
        echo "📝 查看日志..."
        docker compose logs --tail=20
    else
        echo "❌ 服务启动失败！"
        docker compose logs
        exit 1
    fi
else
    echo "❌ 镜像构建失败！"
    exit 1
fi

echo "🎉 部署完成！"
echo "🌐 服务地址: http://localhost:8080"
echo "🏥 健康检查: http://localhost:8080/health"
echo "📋 模型列表: http://localhost:8080/v1/models"
