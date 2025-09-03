# Z2API Deno优化版

## 🎯 项目概述

Z2API Deno优化版是基于原始Z2API项目的TypeScript/Deno实现，提供OpenAI兼容的接口，支持GLM-4.5系列模型。相比原版增加了企业级特性和性能优化。

## ✨ 核心特性

### 🚀 基础功能
- ✅ **OpenAI 兼容接口** - 完全兼容 OpenAI API 格式
- ✅ **多模型支持** - GLM-4.5、GLM-4.5-Thinking、GLM-4.5-Search
- ✅ **流式响应** - 支持流式和非流式响应
- ✅ **思考模式** - 支持模型思考过程展示
- ✅ **搜索功能** - 集成网络搜索能力
- ✅ **匿名 Token** - 自动获取匿名访问令牌

### 🔧 优化特性
- ✅ **性能模式** - 支持fast/balanced/secure三种模式
- ✅ **重试机制** - 指数退避重试策略
- ✅ **结构化日志** - JSON格式日志，支持请求追踪
- ✅ **健康检查** - `/health` 端点提供系统状态
- ✅ **错误处理** - 完善的错误分类和恢复机制
- ✅ **配置管理** - 环境变量驱动的配置系统
- ✅ **CORS支持** - 完整的跨域资源共享支持

## 🏗️ 架构设计

### 配置系统
```typescript
// 环境变量配置
const PORT = parseInt(Deno.env.get("PORT") || "8080");
const DEFAULT_KEY = Deno.env.get("DEFAULT_KEY") || "123456";
const PERFORMANCE_MODE = Deno.env.get("PERFORMANCE_MODE") || "balanced";
const ANON_TOKEN_ENABLED = Deno.env.get("ANON_TOKEN_ENABLED") !== "false";
const THINK_TAGS_MODE = Deno.env.get("THINK_TAGS_MODE") || "think";
```

### 性能模式
| 模式 | 重试次数 | 延迟 | 超时 | 适用场景 |
|------|----------|------|------|----------|
| **fast** | 1次 | 200ms | 10s | 快速响应 |
| **balanced** | 3次 | 1000ms | 30s | 平衡性能 |
| **secure** | 5次 | 2000ms | 60s | 高可靠性 |

## 🚀 快速开始

### 安装Deno
```bash
# macOS/Linux
curl -fsSL https://deno.land/install.sh | sh

# Windows (PowerShell)
irm https://deno.land/install.ps1 | iex
```

### 运行服务
```bash
# 基础运行
deno run --allow-net --allow-env app.ts

# 开发模式（自动重启）
deno run --allow-net --allow-env --watch app.ts

# 生产模式
deno run --allow-net --allow-env --no-check app.ts
```

## ⚙️ 环境配置

### 基础配置
```bash
export PORT=8080                    # 服务端口
export DEFAULT_KEY="your-api-key"   # API认证密钥
export UPSTREAM_TOKEN="your-token"  # 上游API token
```

### 性能配置
```bash
export PERFORMANCE_MODE="balanced"  # 性能模式: fast/balanced/secure
export MAX_RETRIES=3                # 最大重试次数
export RETRY_DELAY=1000             # 重试延迟(ms)
export REQUEST_TIMEOUT=30000        # 请求超时(ms)
export RANDOM_DELAY_MIN=100         # 随机延迟最小值(ms)
export RANDOM_DELAY_MAX=500         # 随机延迟最大值(ms)
```

### 功能配置
```bash
export ANON_TOKEN_ENABLED=true      # 启用匿名token
export THINK_TAGS_MODE="think"      # 思考标签模式: think/strip/raw
export DEBUG_MODE=false             # 调试模式
```

### 日志配置
```bash
export ENABLE_DETAILED_LOGGING=true # 启用详细日志
export LOG_USER_MESSAGES=false      # 记录用户消息
export LOG_RESPONSE_CONTENT=false   # 记录响应内容
```

## 🐳 Docker部署

### 快速开始

**方法一：使用Docker Compose（推荐）**
```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env 文件设置你的API密钥

# 2. 启动服务
docker-compose up -d

# 3. 查看状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f
```

**方法二：使用Docker命令**
```bash
# 构建镜像
docker build -t z2api-deno .

# 运行容器
docker run -d \
  --name z2api-deno \
  -p 8080:8080 \
  -e DEFAULT_KEY=your-api-key \
  -e PERFORMANCE_MODE=balanced \
  z2api-deno
```

### 环境配置

复制 `.env.example` 为 `.env` 并修改配置：
```bash
# 基础配置
DEFAULT_KEY=your-api-key-here
UPSTREAM_TOKEN=your-upstream-token-here

# 性能配置
PERFORMANCE_MODE=balanced    # fast/balanced/secure
MAX_RETRIES=3
REQUEST_TIMEOUT=30000

# 功能配置
ANON_TOKEN_ENABLED=true
THINK_TAGS_MODE=think
DEBUG_MODE=false
```

### 管理命令

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f

# 更新服务
docker-compose pull && docker-compose up -d
```

## 📊 API接口

### 健康检查
```bash
GET /health
```

响应示例：
```json
{
  "status": "ok",
  "timestamp": "2025-01-03T10:00:00.000Z",
  "performance_mode": "balanced",
  "config": {
    "max_retries": 3,
    "retry_delay": 1000,
    "request_timeout": 30000,
    "random_delay": "100-500ms"
  },
  "stats": {
    "total_requests": 1234,
    "average_response_time": 850,
    "error_rate": 2
  }
}
```

### 模型列表
```bash
GET /v1/models
```

### 聊天完成
```bash
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "model": "GLM-4.5",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": true
}
```

## 🔍 日志系统

### 结构化日志格式
```json
{
  "request_id": "req_1234567890abcdef",
  "timestamp": "2025-01-03T10:00:00.000Z",
  "level": "INFO",
  "type": "request",
  "client_ip": "192.168.1.100",
  "api_key": "sk-****",
  "model": "GLM-4.5",
  "user_agent": "curl/7.68.0"
}
```

### 日志级别
- **INFO**: 正常请求和响应
- **WARN**: 警告信息（如重试）
- **ERROR**: 错误信息

## 🔒 安全特性

### API Key保护
- 自动掩码显示（前4后4字符）
- 环境变量存储
- 请求头验证

### 请求伪装
- 随机User-Agent
- 完整浏览器头部伪装
- 防机器人检测

### CORS安全
- 配置跨域访问
- 预检请求处理
- 安全头部设置

## 📈 性能优化

### 重试策略
- 指数退避算法
- 智能错误分类
- 随机延迟防抖

### 内存管理
- 流式处理减少内存占用
- 及时释放资源
- 错误边界保护

### 网络优化
- 连接复用
- 超时控制
- 压缩传输

## 🛠️ 开发指南

### 本地开发
```bash
# 安装开发依赖
deno cache app.ts

# 运行开发服务器
deno run --allow-net --allow-env --watch app.ts

# 代码格式化
deno fmt app.ts

# 代码检查
deno lint app.ts
```

### 测试
```bash
# 健康检查测试
curl http://localhost:8080/health

# API测试
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-key" \
  -d '{"model":"GLM-4.5","messages":[{"role":"user","content":"Hello"}]}'
```

## 🔧 故障排除

### 常见问题

1. **权限错误**
   ```bash
   # 确保给予正确权限
   deno run --allow-net --allow-env app.ts
   ```

2. **端口占用**
   ```bash
   # 检查端口使用
   lsof -i :8080
   # 或更改端口
   export PORT=8081
   ```

3. **网络连接问题**
   ```bash
   # 检查网络连接
   curl -I https://chat.z.ai
   ```

### 调试模式
```bash
export DEBUG_MODE=true
deno run --allow-net --allow-env app.ts
```

## 📋 与原版对比

| 特性 | 原版Z2API | Deno优化版 |
|------|-----------|------------|
| **语言** | Go | TypeScript |
| **类型安全** | ⚠️ | ✅ |
| **性能模式** | ❌ | ✅ |
| **重试机制** | ❌ | ✅ |
| **结构化日志** | ❌ | ✅ |
| **健康检查** | ❌ | ✅ |
| **配置管理** | 硬编码 | 环境变量 |
| **错误处理** | 基础 | 完善 |

## 🎯 适用场景

- 🔬 **原型开发**: 快速验证想法
- 📚 **学习项目**: 现代TypeScript特性
- 🛠️ **开发环境**: 丰富的开发工具支持
- 🌐 **中小型部署**: 适中的性能需求

## 📞 技术支持

如有问题，请：
1. 检查环境配置
2. 查看日志输出
3. 参考故障排除指南
4. 提交Issue反馈
