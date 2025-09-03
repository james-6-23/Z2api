# Z2API Go优化版

## 🎯 项目概述

Z2API Go优化版是基于原始Z2API项目的企业级Go实现，在保持原版核心功能的基础上，增加了完整的并发控制、结构化日志、性能监控等企业级特性。

## ✨ 核心特性

### 🚀 基础功能（继承自原版）
- ✅ **OpenAI 兼容接口** - 完全兼容 OpenAI API 格式
- ✅ **多模型支持** - GLM-4.5、GLM-4.5-Thinking、GLM-4.5-Search
- ✅ **流式响应** - 支持流式和非流式响应
- ✅ **思考模式** - 专业的思考内容处理策略
- ✅ **搜索功能** - 集成网络搜索能力
- ✅ **匿名 Token** - 自动获取匿名访问令牌

### 🔧 企业级优化特性
- ✅ **并发控制** - 信号量机制控制最大并发连接数
- ✅ **性能模式** - 支持fast/balanced/secure三种模式
- ✅ **重试机制** - 指数退避重试策略，智能错误分类
- ✅ **结构化日志** - JSON格式日志，支持请求追踪
- ✅ **健康检查** - `/health` 和 `/status` 端点
- ✅ **内存管理** - 内存使用监控和垃圾回收
- ✅ **流式优化** - 优化的缓冲区策略和连接检测
- ✅ **错误恢复** - 完善的错误处理和恢复机制
- ✅ **配置管理** - 环境变量驱动的配置系统

## 🏗️ 架构设计

### 并发控制架构
```go
// 全局并发控制
var (
    maxConcurrentConnections = 1000
    currentConnections      int64
    connectionSemaphore     chan struct{}
)

// 中间件实现
func concurrencyControlMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        select {
        case connectionSemaphore <- struct{}{}:
            // 获取连接许可
            atomic.AddInt64(&currentConnections, 1)
            defer func() {
                <-connectionSemaphore
                atomic.AddInt64(&currentConnections, -1)
            }()
            next(w, r)
        default:
            // 连接数已满，拒绝请求
            http.Error(w, "Server too busy", http.StatusServiceUnavailable)
        }
    }
}
```

### 性能模式配置
| 模式 | 重试次数 | 延迟 | 超时 | 适用场景 |
|------|----------|------|------|----------|
| **fast** | 1次 | 200ms | 10s/60s | 快速响应 |
| **balanced** | 3次 | 1000ms | 120s/300s | 平衡性能 |
| **secure** | 5次 | 2000ms | 60s/600s | 高可靠性 |

## 🚀 快速开始

### 编译和运行
```bash
# 克隆项目
git clone <repository-url>
cd z2api-go-optimized

# 编译
go build -o z2api-optimized main.go

# 运行
./z2api-optimized
```

### 开发模式
```bash
# 直接运行
go run main.go

# 启用调试模式
DEBUG_MODE=true go run main.go
```

## ⚙️ 环境配置

### 基础配置
```bash
export PORT=8080                    # 服务端口
export DEFAULT_KEY="your-api-key"   # API认证密钥
export UPSTREAM_URL="https://chat.z.ai/api/chat/completions"
export UPSTREAM_TOKEN="your-token"  # 上游API token
```

### 性能配置
```bash
export PERFORMANCE_MODE="balanced"  # 性能模式: fast/balanced/secure
export MAX_RETRIES=3                # 最大重试次数
export RETRY_DELAY=1000             # 重试延迟(ms)
export REQUEST_TIMEOUT=120000       # 请求超时(ms)
export STREAM_TIMEOUT=300000        # 流式超时(ms)
export RANDOM_DELAY_MIN=100         # 随机延迟最小值(ms)
export RANDOM_DELAY_MAX=500         # 随机延迟最大值(ms)
```

### 并发控制配置
```bash
export MAX_CONCURRENT_CONNECTIONS=1000  # 最大并发连接数
export CONNECTION_QUEUE_SIZE=500        # 连接队列大小
export MAX_CONNECTION_TIME=600000       # 最大连接时间(ms)
export MEMORY_LIMIT_MB=2048             # 内存限制(MB)
```

### 流处理优化配置
```bash
export STREAM_BUFFER_SIZE=16384         # 流缓冲区大小(bytes)
export DISABLE_CONNECTION_CHECK=false   # 禁用连接检测
export CONNECTION_CHECK_INTERVAL=20     # 连接检测间隔
```

### 功能配置
```bash
export ANON_TOKEN_ENABLED=true      # 启用匿名token
export THINK_TAGS_MODE="think"      # 思考标签模式: think/strip/raw
export DEBUG_MODE=false             # 调试模式
export ENABLE_METRICS=true          # 启用性能监控
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
docker build -t z2api-go .

# 运行容器
docker run -d \
  --name z2api-go \
  -p 8080:8080 \
  -e DEFAULT_KEY=your-api-key \
  -e PERFORMANCE_MODE=balanced \
  -e MAX_CONCURRENT_CONNECTIONS=1000 \
  z2api-go
```

### 环境配置

复制 `.env.example` 为 `.env` 并修改配置：
```bash
# 基础配置
DEFAULT_KEY=your-api-key-here
UPSTREAM_TOKEN=your-upstream-token-here

# 性能配置
PERFORMANCE_MODE=balanced         # fast/balanced/secure
MAX_CONCURRENT_CONNECTIONS=1000   # 最大并发连接数
REQUEST_TIMEOUT=120000           # 请求超时(ms)

# 功能配置
ANON_TOKEN_ENABLED=true
THINK_TAGS_MODE=think
ENABLE_METRICS=true
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

# 查看资源使用
docker stats z2api-go
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
  "timestamp": "2025-01-03T10:00:00Z",
  "version": "2.1.0",
  "build_date": "2025-01-03",
  "description": "Z2API Go优化版 - 基于原版Z2API的企业级优化实现",
  "performance_mode": "balanced",
  "config": {
    "max_retries": 3,
    "retry_delay": 1000,
    "request_timeout": 120000,
    "random_delay": "100-500ms",
    "max_concurrent_connections": 1000,
    "stream_buffer_size": 16384,
    "connection_check_enabled": true
  },
  "stats": {
    "total_requests": 1234,
    "average_response_time": 850,
    "error_rate": 2,
    "current_connections": 45
  },
  "improvements": [
    "基于原版Z2API的企业级优化",
    "完整的并发控制机制",
    "结构化日志系统",
    "性能模式配置",
    "重试机制和错误恢复",
    "健康检查和监控",
    "匿名token支持",
    "专业思考内容处理"
  ]
}
```

### 系统状态
```bash
GET /status
```

响应示例：
```json
{
  "current_connections": 45,
  "max_connections": 1000,
  "memory_usage_mb": 128,
  "memory_limit_mb": 2048,
  "total_requests": 1234,
  "error_count": 12,
  "uptime_seconds": 3600
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
  "model": "GLM-4.5-Thinking",
  "messages": [
    {"role": "user", "content": "解释一下量子计算的原理"}
  ],
  "stream": true
}
```

## 🔍 日志系统

### 结构化日志格式
```json
{
  "request_id": "req_1234567890abcdef",
  "timestamp": "2025-01-03T10:00:00Z",
  "level": "INFO",
  "type": "request",
  "client_ip": "192.168.1.100",
  "api_key": "sk-****",
  "model": "GLM-4.5-Thinking",
  "user_agent": "curl/7.68.0",
  "parameters": {
    "message_count": 1,
    "parameters": {
      "stream": true,
      "temperature": null,
      "max_tokens": null
    }
  }
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

### 并发安全
- 原子操作保护共享变量
- 信号量控制并发数
- 连接状态检测

### 内存安全
- 缓冲区大小限制
- 内存使用监控
- 自动垃圾回收

## 📈 性能优化

### 并发控制
- 信号量机制限制并发数
- 连接队列管理
- 智能拒绝策略

### 流式优化
- 可配置缓冲区大小
- 智能连接检测
- 内存泄漏防护

### 重试策略
- 指数退避算法
- 智能错误分类
- 随机延迟防抖

## 🛠️ 开发指南

### 本地开发
```bash
# 安装依赖
go mod init z2api-optimized
go mod tidy

# 运行开发服务器
go run main.go

# 代码格式化
go fmt main.go

# 代码检查
go vet main.go
```

### 测试
```bash
# 健康检查测试
curl http://localhost:8080/health

# 系统状态测试
curl http://localhost:8080/status

# API测试
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-key" \
  -d '{"model":"GLM-4.5","messages":[{"role":"user","content":"Hello"}]}'
```

### 性能测试
```bash
# 并发测试
ab -n 1000 -c 50 -H "Authorization: Bearer your-key" \
  -p test.json -T application/json \
  http://localhost:8080/v1/chat/completions
```

## 🔧 故障排除

### 常见问题

1. **连接数过多**
   ```bash
   # 检查当前连接数
   curl http://localhost:8080/status
   
   # 调整最大连接数
   export MAX_CONCURRENT_CONNECTIONS=2000
   ```

2. **内存使用过高**
   ```bash
   # 检查内存使用
   curl http://localhost:8080/status
   
   # 调整内存限制
   export MEMORY_LIMIT_MB=4096
   ```

3. **响应超时**
   ```bash
   # 调整超时设置
   export REQUEST_TIMEOUT=180000
   export STREAM_TIMEOUT=600000
   ```

### 调试模式
```bash
export DEBUG_MODE=true
./z2api-optimized
```

## 📋 与原版对比

| 特性 | 原版Z2API | Go优化版 |
|------|-----------|----------|
| **并发控制** | ❌ | ✅ 信号量机制 |
| **性能模式** | ❌ | ✅ 3种模式 |
| **重试机制** | ❌ | ✅ 指数退避 |
| **结构化日志** | ❌ | ✅ JSON格式 |
| **健康检查** | ❌ | ✅ 多端点 |
| **内存管理** | 基础 | ✅ 监控+GC |
| **流式优化** | 基础 | ✅ 缓冲+检测 |
| **错误处理** | 基础 | ✅ 分层处理 |
| **配置管理** | 硬编码 | ✅ 环境变量 |
| **匿名token** | ✅ | ✅ 保持 |
| **思考处理** | ✅ | ✅ 保持 |

## 🎯 适用场景

- 🏭 **生产环境**: 高并发、高可靠性要求
- 📊 **企业应用**: 需要完整监控和日志
- 🚀 **高性能场景**: 大规模部署
- 🔐 **安全要求**: 需要访问控制和审计

## 📞 技术支持

如有问题，请：
1. 检查健康检查端点状态
2. 查看结构化日志输出
3. 参考故障排除指南
4. 提交Issue反馈

## 🎉 改进亮点

相比原版Z2API，本优化版本提供：

1. **企业级稳定性** - 并发控制和错误恢复
2. **完整可观测性** - 结构化日志和性能监控
3. **灵活配置管理** - 环境变量驱动配置
4. **优化的性能** - 多种性能模式和流式优化
5. **保持兼容性** - 完全保持原版的核心功能
