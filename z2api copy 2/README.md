# Z2API - Z.AI API 代理服务

## 🎯 项目概述

Z2API 是一个 Z.AI API 代理服务，提供 OpenAI 兼容的接口，支持 GLM-4.5 系列模型。所有配置参数都硬编码在源代码中，无需外部配置文件。

## ✨ 核心特性

- ✅ **OpenAI 兼容接口** - 完全兼容 OpenAI API 格式
- ✅ **多模型支持** - GLM-4.5、GLM-4.5-Thinking、GLM-4.5-Search
- ✅ **流式响应** - 支持流式和非流式响应
- ✅ **思考模式** - 支持模型思考过程展示
- ✅ **搜索功能** - 集成网络搜索能力
- ✅ **匿名 Token** - 自动获取匿名访问令牌
- ✅ **Docker 部署** - 完整的容器化部署方案

## 🏗️ 架构设计

### 硬编码配置
```go
const (
    UpstreamUrl       = "https://chat.z.ai/api/chat/completions"
    DefaultKey        = "sk-tbkFoKzk9a531YyUNNF5"
    DefaultModelName  = "GLM-4.5"
    ThinkingModelName = "GLM-4.5-Thinking"
    SearchModelName   = "GLM-4.5-Search"
    Port              = ":8080"
    DebugMode         = true
)
```

### 支持的模型
- **GLM-4.5** - 基础对话模型
- **GLM-4.5-Thinking** - 带思考过程的模型
- **GLM-4.5-Search** - 带网络搜索的模型

## 🚀 快速部署

### 方法一：使用部署脚本（推荐）

```bash
# 1. 进入项目目录
cd z2api

# 2. 给脚本执行权限
chmod +x deploy.sh

# 3. 运行部署脚本
./deploy.sh
```

### 方法二：手动 Docker Compose 部署

```bash
# 1. 进入项目目录
cd z2api

# 2. 构建和启动服务
docker compose up -d --build

# 3. 查看服务状态
docker compose ps

# 4. 查看日志
docker compose logs -f
```

### 方法三：手动 Docker 构建

```bash
# 1. 构建镜像
docker build -t z2api:latest .

# 2. 运行容器
docker run -d \
  --name z2api-service \
  -p 8080:8080 \
  --restart unless-stopped \
  z2api:latest

# 3. 查看日志
docker logs -f z2api-service
```

## 🧪 API 测试

### 健康检查
```bash
curl http://localhost:8080/v1/models
```

### 获取模型列表
```bash
curl -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
     http://localhost:8080/v1/models
```

### 基础对话测试
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
  -d '{
    "model": "GLM-4.5",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": false,
    "max_tokens": 100
  }'
```

### 思考模式测试
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
  -d '{
    "model": "GLM-4.5-Thinking",
    "messages": [{"role": "user", "content": "解释一下量子计算的原理"}],
    "stream": false,
    "max_tokens": 500
  }'
```

### 搜索模式测试
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
  -d '{
    "model": "GLM-4.5-Search",
    "messages": [{"role": "user", "content": "今天的新闻有什么"}],
    "stream": false,
    "max_tokens": 300
  }'
```

### 流式响应测试
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
  -d '{
    "model": "GLM-4.5",
    "messages": [{"role": "user", "content": "写一首诗"}],
    "stream": true,
    "max_tokens": 200
  }'
```

## 🔧 服务管理

### 查看服务状态
```bash
# Docker Compose 方式
docker compose ps

# 直接 Docker 方式
docker ps | grep z2api
```

### 查看日志
```bash
# Docker Compose 方式
docker compose logs -f

# 直接 Docker 方式
docker logs -f z2api-service
```

### 重启服务
```bash
# Docker Compose 方式
docker compose restart

# 直接 Docker 方式
docker restart z2api-service
```

### 停止服务
```bash
# Docker Compose 方式
docker compose down

# 直接 Docker 方式
docker stop z2api-service
docker rm z2api-service
```

### 进入容器
```bash
# Docker Compose 方式
docker compose exec z2api sh

# 直接 Docker 方式
docker exec -it z2api-service sh
```

## 📊 配置说明

### 硬编码配置项
| 配置项 | 值 | 说明 |
|--------|-----|------|
| **端口** | 8080 | 服务监听端口 |
| **API 密钥** | sk-tbkFoKzk9a531YyUNNF5 | 客户端认证密钥 |
| **上游 URL** | https://chat.z.ai/api/chat/completions | Z.AI API 地址 |
| **调试模式** | true | 启用详细日志 |
| **匿名 Token** | true | 启用匿名令牌获取 |

### 模型配置
| 模型名称 | 功能 | 上游模型 |
|----------|------|----------|
| GLM-4.5 | 基础对话 | 0727-360B-API |
| GLM-4.5-Thinking | 思考模式 | 0727-360B-API + thinking |
| GLM-4.5-Search | 搜索模式 | 0727-360B-API + web_search |

## 🛠️ 故障排除

### 常见问题

#### 1. 服务无法启动
```bash
# 检查端口占用
netstat -tuln | grep :8080

# 查看详细错误
docker compose logs
```

#### 2. API 请求失败
```bash
# 检查服务健康状态
curl http://localhost:8080/v1/models

# 验证 API 密钥
curl -H "Authorization: Bearer sk-tbkFoKzk9a531YyUNNF5" \
     http://localhost:8080/v1/models
```

#### 3. 上游连接失败
```bash
# 检查网络连接
curl -I https://chat.z.ai

# 查看详细日志
docker compose logs -f | grep "上游"
```

#### 4. 容器构建失败
```bash
# 清理 Docker 缓存
docker system prune -f

# 重新构建
docker compose build --no-cache
```

## 📋 技术规格

- **语言**: Go 1.21
- **框架**: 标准库 net/http
- **容器**: Docker + Alpine Linux
- **架构**: 多阶段构建
- **端口**: 8080
- **健康检查**: /v1/models 端点
- **日志**: 结构化 JSON 日志

## 🔗 相关链接

- [Z.AI 官网](https://chat.z.ai)
- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [Docker 官方文档](https://docs.docker.com/)

---

**🎉 Z2API 部署完成！开始使用您的 Z.AI API 代理服务吧！**
