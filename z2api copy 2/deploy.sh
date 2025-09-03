#!/bin/bash

# Z2API 部署脚本
# 用于快速构建和部署 Z2API 服务

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}🚀 Z2API 部署脚本${NC}"
echo "=================================="

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装，请先安装 Docker${NC}"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker 环境检查通过${NC}"

# 检查必要文件
echo ""
echo -e "${CYAN}📋 检查必要文件${NC}"

required_files=("main.go" "go.mod" "Dockerfile" "docker-compose.yml")
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ $file${NC}"
    else
        echo -e "${RED}❌ $file 文件缺失${NC}"
        exit 1
    fi
done

# 检查端口占用
echo ""
echo -e "${CYAN}🔍 检查端口占用${NC}"
if netstat -tuln 2>/dev/null | grep -q ":8080 "; then
    echo -e "${YELLOW}⚠️ 端口 8080 已被占用${NC}"
    echo "是否继续部署？(y/n)"
    read -p "> " continue_deploy
    if [ "$continue_deploy" != "y" ] && [ "$continue_deploy" != "Y" ]; then
        echo -e "${YELLOW}部署已取消${NC}"
        exit 0
    fi
else
    echo -e "${GREEN}✅ 端口 8080 可用${NC}"
fi

# 停止现有容器
echo ""
echo -e "${CYAN}🛑 停止现有容器${NC}"
docker compose down 2>/dev/null || echo "无现有容器需要停止"

# 构建和启动服务
echo ""
echo -e "${CYAN}🔨 构建和启动服务${NC}"
echo "开始构建 Docker 镜像..."

if docker compose up -d --build; then
    echo -e "${GREEN}✅ 服务启动成功${NC}"
else
    echo -e "${RED}❌ 服务启动失败${NC}"
    echo "查看错误日志："
    docker compose logs
    exit 1
fi

# 等待服务启动
echo ""
echo -e "${CYAN}⏳ 等待服务启动${NC}"
sleep 10

# 健康检查
echo ""
echo -e "${CYAN}🏥 健康检查${NC}"

max_attempts=30
attempt=1

while [ $attempt -le $max_attempts ]; do
    if curl -s -f http://localhost:8080/v1/models >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 服务健康检查通过${NC}"
        break
    else
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    fi
done

if [ $attempt -gt $max_attempts ]; then
    echo -e "${RED}❌ 服务健康检查失败${NC}"
    echo "查看服务日志："
    docker compose logs --tail=20
    exit 1
fi

# 显示服务信息
echo ""
echo -e "${CYAN}📊 服务信息${NC}"
echo "=================================="
echo -e "服务名称: ${GREEN}Z2API${NC}"
echo -e "服务地址: ${GREEN}http://localhost:8080${NC}"
echo -e "API 密钥: ${GREEN}sk-tbkFoKzk9a531YyUNNF5${NC}"
echo -e "容器状态: ${GREEN}$(docker compose ps --format 'table {{.Service}}\t{{.Status}}')"

# API 测试
echo ""
echo -e "${CYAN}🧪 API 测试${NC}"

echo "1. 测试模型列表："
if curl -s http://localhost:8080/v1/models | jq . >/dev/null 2>&1; then
    echo -e "${GREEN}✅ 模型列表 API 正常${NC}"
    curl -s http://localhost:8080/v1/models | jq '.data[].id' 2>/dev/null || echo "模型列表获取成功"
else
    echo -e "${YELLOW}⚠️ 模型列表 API 测试失败（可能是 jq 未安装）${NC}"
    curl -s http://localhost:8080/v1/models
fi

echo ""
echo "2. 测试聊天 API："
chat_response=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
  -d '{
    "model": "GLM-4.5",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false,
    "max_tokens": 10
  }')

if echo "$chat_response" | grep -q "choices"; then
    echo -e "${GREEN}✅ 聊天 API 正常${NC}"
else
    echo -e "${YELLOW}⚠️ 聊天 API 测试可能有问题${NC}"
    echo "响应: $chat_response"
fi

# 显示管理命令
echo ""
echo -e "${CYAN}🔧 管理命令${NC}"
echo "=================================="
echo "查看日志: docker compose logs -f"
echo "重启服务: docker compose restart"
echo "停止服务: docker compose down"
echo "查看状态: docker compose ps"
echo "进入容器: docker compose exec z2api sh"

echo ""
echo -e "${GREEN}🎉 Z2API 部署完成！${NC}"
echo -e "访问地址: ${GREEN}http://localhost:8080${NC}"
echo -e "API 文档: ${GREEN}OpenAI 兼容接口${NC}"
