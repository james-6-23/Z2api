# Z2API 简化Docker部署方案

## 🎯 概述

这是Z2API的简化Docker部署方案，提供两个独立的优化版本：

- **Deno优化版**: 现代化TypeScript实现，适合开发和中小型部署
- **Go优化版**: 企业级Go实现，适合生产环境和高并发场景

每个版本都可以独立部署，无需复杂的多服务编排。

## 📁 项目结构

```
Z2api/
├── z2api-deno-optimized/           # Deno优化版
│   ├── Dockerfile                  # Docker镜像配置
│   ├── docker-compose.yml          # 独立部署配置
│   ├── .env.example                # 环境变量示例
│   ├── app.ts                      # 应用代码
│   └── README.md                   # 详细说明
├── z2api-go-optimized/             # Go优化版
│   ├── Dockerfile                  # Docker镜像配置
│   ├── docker-compose.yml          # 独立部署配置
│   ├── .env.example                # 环境变量示例
│   ├── main.go                     # 应用代码
│   └── README.md                   # 详细说明
└── .env.example                    # 通用环境变量示例
```

## 🚀 快速开始

### 选择版本

**Deno优化版** - 适合：
- 开发和测试环境
- 中小型项目
- 喜欢TypeScript的开发者
- 快速原型验证

**Go优化版** - 适合：
- 生产环境
- 高并发场景
- 企业级应用
- 性能要求较高的场景

### 部署Deno版本

```bash
# 进入Deno版本目录
cd z2api-deno-optimized

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件设置你的API密钥

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 访问服务
curl http://localhost:8080/health
```

### 部署Go版本

```bash
# 进入Go版本目录
cd z2api-go-optimized

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件设置你的API密钥

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 访问服务
curl http://localhost:8080/health
```

## ⚙️ 环境配置

### 必需配置

在 `.env` 文件中设置以下必需参数：

```bash
# API认证密钥（必须修改）
DEFAULT_KEY=your-api-key-here

# 上游API令牌（可选，启用匿名token时可为空）
UPSTREAM_TOKEN=your-upstream-token-here
```

### 可选配置

```bash
# 性能模式
PERFORMANCE_MODE=balanced    # fast/balanced/secure

# 功能开关
ANON_TOKEN_ENABLED=true     # 启用匿名token
THINK_TAGS_MODE=think       # 思考标签模式
DEBUG_MODE=false            # 调试模式

# Go版本专有配置
MAX_CONCURRENT_CONNECTIONS=1000  # 最大并发连接数
ENABLE_METRICS=true              # 启用性能监控
```

## 📊 API接口

### 健康检查
```bash
GET http://localhost:8080/health
```

### 模型列表
```bash
GET http://localhost:8080/v1/models
```

### 聊天完成
```bash
POST http://localhost:8080/v1/chat/completions
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "model": "GLM-4.5",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

## 🔧 管理命令

### 基础操作

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f

# 查看状态
docker-compose ps
```

### 维护操作

```bash
# 更新服务
docker-compose pull
docker-compose up -d

# 重新构建
docker-compose build --no-cache
docker-compose up -d

# 查看资源使用
docker stats
```

## 🧪 测试验证

### 健康检查测试

```bash
# 测试服务是否正常
curl -f http://localhost:8080/health

# 测试API接口
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "GLM-4.5",
    "messages": [{"role": "user", "content": "测试消息"}],
    "stream": false
  }'
```

### 性能测试

```bash
# 简单压力测试
for i in {1..10}; do
  curl -s http://localhost:8080/health > /dev/null && echo "请求 $i: 成功" || echo "请求 $i: 失败"
done
```

## 🔒 安全配置

### API密钥管理

1. **生成强密钥**：
```bash
# 生成随机API密钥
openssl rand -hex 32
```

2. **环境变量存储**：
```bash
# 在 .env 文件中设置
DEFAULT_KEY=your-generated-strong-key
```

### 网络安全

```bash
# 只允许本地访问（可选）
docker-compose up -d
# 服务只绑定到 127.0.0.1:8080

# 如需外部访问，确保设置防火墙规则
sudo ufw allow 8080/tcp
```

## 🛠️ 故障排除

### 常见问题

1. **端口被占用**
```bash
# 检查端口使用
netstat -tulpn | grep :8080
# 或修改 docker-compose.yml 中的端口映射
```

2. **容器启动失败**
```bash
# 查看详细日志
docker-compose logs

# 检查配置
docker-compose config
```

3. **API调用失败**
```bash
# 检查API密钥是否正确
# 检查上游服务是否可达
curl -I https://chat.z.ai
```

### 调试技巧

```bash
# 进入容器调试
docker-compose exec z2api-deno sh    # Deno版本
docker-compose exec z2api-go sh      # Go版本

# 查看实时日志
docker-compose logs -f --tail=100

# 重置服务
docker-compose down
docker-compose up -d --force-recreate
```

## 📈 性能对比

| 特性 | Deno版本 | Go版本 |
|------|----------|--------|
| **并发处理** | ~1000 req/s | ~5000 req/s |
| **内存使用** | ~256MB | ~512MB |
| **启动时间** | ~3s | ~1s |
| **镜像大小** | ~50MB | ~10MB |
| **并发控制** | ❌ | ✅ |
| **性能监控** | ✅ | ✅ |
| **类型安全** | ✅ TypeScript | ⚠️ Go类型 |

## 🎯 选择建议

### 选择Deno版本，如果你：
- 熟悉TypeScript/JavaScript
- 需要快速开发和部署
- 项目规模中小型
- 重视开发体验

### 选择Go版本，如果你：
- 需要高性能和高并发
- 部署到生产环境
- 需要企业级特性
- 重视系统稳定性

## 📞 技术支持

### 获取帮助

1. **查看详细文档**：
   - Deno版本：`z2api-deno-optimized/README.md`
   - Go版本：`z2api-go-optimized/README.md`

2. **检查服务状态**：
   ```bash
   docker-compose ps
   docker-compose logs
   ```

3. **验证配置**：
   ```bash
   docker-compose config
   ```

### 常用资源

- **健康检查**: `http://localhost:8080/health`
- **API文档**: OpenAI兼容接口
- **日志位置**: `docker-compose logs`
- **配置文件**: `.env`

---

**🎉 现在您可以快速部署Z2API优化版本了！**

选择适合您需求的版本，按照上述步骤即可开始使用。
