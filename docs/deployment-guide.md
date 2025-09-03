# Z2API Deployment Guide

## Overview

This guide covers deployment strategies for all three versions of the API proxy projects:

- **Deno Version**: TypeScript-based DeepInfra proxy
- **Go Version**: Optimized Go-based DeepInfra proxy  
- **Z2API Main**: Go-based Z.ai proxy

## Quick Start

### Z2API Main Project (Recommended)

```bash
# Clone repository
git clone <repository-url>
cd Z2api/z2api

# Build binary
go build -o z2api main.go

# Run with default settings
./z2api
```

### Go Version (Production Ready)

```bash
cd Z2api/go-version

# Build optimized binary
go build -ldflags="-s -w" -o go-proxy main.go

# Run with configuration
export MAX_CONCURRENT_CONNECTIONS=1000
export ENABLE_DETAILED_LOGGING=true
./go-proxy
```

### Deno Version

```bash
cd Z2api/deno-version

# Install Deno (if not installed)
curl -fsSL https://deno.land/install.sh | sh

# Run with permissions
deno run --allow-net --allow-env app.ts
```

## Environment Configuration

### Z2API Main Project

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:8080` | Server port |
| `UPSTREAM_URL` | `https://chat.z.ai/api/chat/completions` | Z.ai API endpoint |
| `DEFAULT_KEY` | `123456` | API authentication key |
| `ANON_TOKEN_ENABLED` | `true` | Enable anonymous token fetching |
| `THINK_TAGS_MODE` | `think` | Thinking content processing mode |

### Go Version

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Server port |
| `PERFORMANCE_MODE` | `balanced` | Performance mode (fast/balanced/secure) |
| `MAX_CONCURRENT_CONNECTIONS` | `1000` | Maximum concurrent connections |
| `ENABLE_DETAILED_LOGGING` | `true` | Enable structured logging |
| `STREAM_BUFFER_SIZE` | `16384` | Stream buffer size in bytes |

### Deno Version

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Server port |
| `PERFORMANCE_MODE` | `balanced` | Performance mode |
| `DEEPINFRA_MIRRORS` | - | Comma-separated mirror endpoints |
| `VALID_API_KEYS` | `linux.do` | Valid API keys |

## Docker Deployment

### Z2API Main Project

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o z2api main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/z2api .
EXPOSE 8080
CMD ["./z2api"]
```

```bash
# Build and run
docker build -t z2api .
docker run -p 8080:8080 -e DEFAULT_KEY=your-key z2api
```

### Go Version

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go-version/ .
RUN go mod init go-proxy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-proxy main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/go-proxy .
EXPOSE 8000
CMD ["./go-proxy"]
```

### Deno Version

```dockerfile
FROM denoland/deno:alpine

WORKDIR /app
COPY deno-version/app.ts .

EXPOSE 8000
CMD ["run", "--allow-net", "--allow-env", "app.ts"]
```

## Production Deployment

### Using Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  z2api:
    build: 
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DEFAULT_KEY=${API_KEY}
      - ANON_TOKEN_ENABLED=true
      - THINK_TAGS_MODE=think
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  go-proxy:
    build:
      context: .
      dockerfile: Dockerfile.go
    ports:
      - "8000:8000"
    environment:
      - PERFORMANCE_MODE=secure
      - MAX_CONCURRENT_CONNECTIONS=2000
      - ENABLE_DETAILED_LOGGING=true
    restart: unless-stopped
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: z2api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: z2api
  template:
    metadata:
      labels:
        app: z2api
    spec:
      containers:
      - name: z2api
        image: z2api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DEFAULT_KEY
          valueFrom:
            secretKeyRef:
              name: z2api-secret
              key: api-key
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: z2api-service
spec:
  selector:
    app: z2api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Reverse Proxy Configuration

### Nginx

```nginx
# /etc/nginx/sites-available/z2api
upstream z2api_backend {
    server 127.0.0.1:8080;
    # Add more servers for load balancing
    # server 127.0.0.1:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://z2api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # For streaming responses
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }
}
```

### Caddy

```caddyfile
# Caddyfile
your-domain.com {
    reverse_proxy localhost:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
    }
}
```

## Monitoring and Logging

### Health Check Endpoints

- **Z2API**: Currently no health endpoint (needs implementation)
- **Go Version**: `GET /health` - Returns system status
- **Deno Version**: `GET /health` - Returns performance metrics

### Log Configuration

```bash
# Enable detailed logging
export ENABLE_DETAILED_LOGGING=true

# Log to file
./z2api 2>&1 | tee -a /var/log/z2api.log

# JSON log parsing with jq
tail -f /var/log/z2api.log | jq '.'
```

### Prometheus Metrics (Go Version)

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'z2api'
    static_configs:
      - targets: ['localhost:8000']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Performance Tuning

### System Limits

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Kernel parameters
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65536" >> /etc/sysctl.conf
sysctl -p
```

### Go Version Optimization

```bash
# Environment variables for production
export PERFORMANCE_MODE=secure
export MAX_CONCURRENT_CONNECTIONS=2000
export STREAM_BUFFER_SIZE=32768
export REQUEST_TIMEOUT=120000
export STREAM_TIMEOUT=600000
```

### Z2API Optimization

```bash
# Recommended settings
export ANON_TOKEN_ENABLED=true
export THINK_TAGS_MODE=think
# Add more configuration options after implementing them
```

## Security Considerations

### API Key Management

```bash
# Use environment variables
export DEFAULT_KEY="$(openssl rand -hex 32)"

# Or use Docker secrets
echo "your-secure-key" | docker secret create api_key -
```

### Network Security

```bash
# Firewall rules (UFW)
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 8080/tcp  # Block direct access
ufw enable
```

### SSL/TLS Configuration

```nginx
# SSL configuration for Nginx
server {
    listen 443 ssl http2;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
}
```

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   lsof -i :8080
   kill -9 <PID>
   ```

2. **Permission denied**
   ```bash
   sudo setcap 'cap_net_bind_service=+ep' ./z2api
   ```

3. **Memory issues**
   ```bash
   # Monitor memory usage
   top -p $(pgrep z2api)
   ```

### Debug Mode

```bash
# Enable debug logging
export DEBUG_MODE=true
./z2api
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup environment variables
env | grep -E "(DEFAULT_KEY|UPSTREAM_TOKEN)" > .env.backup

# Backup binary
cp z2api z2api.backup.$(date +%Y%m%d)
```

### Log Rotation

```bash
# Setup logrotate
cat > /etc/logrotate.d/z2api << EOF
/var/log/z2api.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF
```
