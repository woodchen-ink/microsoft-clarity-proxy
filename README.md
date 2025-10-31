# Microsoft Clarity Proxy

[![Build and Push Docker Image](https://github.com/YOUR_USERNAME/microsoft-clarity/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/YOUR_USERNAME/microsoft-clarity/actions/workflows/docker-publish.yml)
[![Docker Image](https://ghcr-badge.egpl.dev/YOUR_USERNAME/microsoft-clarity/latest_tag?trim=major&label=latest)](https://github.com/YOUR_USERNAME/microsoft-clarity/pkgs/container/microsoft-clarity)

一个用于反向代理 Microsoft Clarity 统计服务的 Go 程序，解决 CORS 跨域问题并绕过广告拦截插件。

## 特性

✅ **完全解决 CORS 问题** - 自动处理跨域请求头，支持带凭证的请求
✅ **自动 URL 替换** - 将所有 Clarity 域名替换为你的代理域名
✅ **避免广告拦截** - 使用单字母路径 (`/ms/`) 避免被识别
✅ **多平台支持** - 支持 amd64 和 arm64 架构
✅ **Docker 部署** - 开箱即用的 Docker 镜像
✅ **健康检查** - 内置健康检查端点

## 快速开始

### 使用 Docker（推荐）

```bash
docker run -d \
  --name clarity-proxy \
  -p 8081:8081 \
  -e PROXY_DOMAIN=https://analytics.example.com \
  ghcr.io/YOUR_USERNAME/microsoft-clarity:latest
```

### 使用 Docker Compose

1. 创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  clarity-proxy:
    image: ghcr.io/YOUR_USERNAME/microsoft-clarity:latest
    container_name: clarity-proxy
    restart: unless-stopped
    ports:
      - "8081:8081"
    environment:
      - PROXY_DOMAIN=https://analytics.example.com
      - LISTEN_PORT=8081
```

2. 启动服务:

```bash
docker-compose up -d
```

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/YOUR_USERNAME/microsoft-clarity.git
cd microsoft-clarity

# 编译
go build -o clarity-proxy clarity-proxy.go

# 运行
PROXY_DOMAIN=https://analytics.example.com ./clarity-proxy
```

## 环境变量

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `PROXY_DOMAIN` | 你的代理域名 | `https://analytics.czl.net` | 是 |
| `LISTEN_PORT` | 监听端口 | `8081` | 否 |

## Nginx 配置

将以下配置添加到你的 nginx server 块中：

```nginx
server {
    listen 443 ssl http2;
    server_name analytics.example.com;

    # SSL 配置...

    location /ms/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Origin $http_origin;
        proxy_set_header Referer $http_referer;

        proxy_pass_request_headers on;
        proxy_pass_request_body on;

        # 禁用缓冲和压缩,避免 HTTP/2 协议错误
        proxy_buffering off;
        proxy_request_buffering off;
        gzip off;
    }
}
```

重载 nginx:

```bash
nginx -t && nginx -s reload
```

## 前端集成

在你的网站中添加以下代码：

```html
<script type="text/javascript">
    (function(c,l,a,r,i,t,y){
        c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};
        t=l.createElement(r);t.async=1;
        // 使用你的代理域名和项目 ID
        t.src="https://analytics.example.com/ms/t.js?id=YOUR_PROJECT_ID";
        y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y);
    })(window, document, "clarity", "script", "YOUR_PROJECT_ID");
</script>
```

将 `analytics.example.com` 替换为你的代理域名，`YOUR_PROJECT_ID` 替换为你的 Clarity 项目 ID。

## 路径映射

| 前端请求 | 转发到 |
|---------|--------|
| `/ms/t.js?id=xxx` | `https://www.clarity.ms/tag/xxx` |
| `/ms/j/0.8.38/clarity.js` | `https://scripts.clarity.ms/0.8.38/clarity.js` |
| `/ms/i.gif` | `https://c.clarity.ms/c.gif` |
| `/ms/p` | `https://k.clarity.ms/collect` |
| `/ms/c/*` | `https://c.clarity.ms/*` |

## 健康检查

访问 `/health` 端点查看服务状态：

```bash
curl http://localhost:8081/health
```

响应示例：

```json
{
  "status": "ok",
  "service": "clarity-proxy",
  "domain": "https://analytics.example.com"
}
```

## 开发

### 本地运行

```bash
go run clarity-proxy.go
```

### 构建 Docker 镜像

```bash
docker build -t clarity-proxy .
```

### 运行测试

```bash
go test ./...
```

## 部署

### 使用 systemd (Linux)

1. 创建 systemd 服务文件：

```bash
sudo nano /etc/systemd/system/clarity-proxy.service
```

2. 添加以下内容：

```ini
[Unit]
Description=Clarity Proxy Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/clarity-proxy
Environment="PROXY_DOMAIN=https://analytics.example.com"
Environment="LISTEN_PORT=8081"
ExecStart=/opt/clarity-proxy/clarity-proxy
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

3. 启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable clarity-proxy
sudo systemctl start clarity-proxy
sudo systemctl status clarity-proxy
```

### 使用 Docker Swarm

```bash
docker service create \
  --name clarity-proxy \
  --publish 8081:8081 \
  --env PROXY_DOMAIN=https://analytics.example.com \
  --replicas 3 \
  ghcr.io/YOUR_USERNAME/microsoft-clarity:latest
```

### 使用 Kubernetes

查看 [k8s-deployment.yaml](k8s-deployment.yaml) 示例配置。

## 故障排查

### 问题 1: CORS 错误

确保 `PROXY_DOMAIN` 环境变量设置正确，并且与前端代码中的域名一致。

### 问题 2: 502 Bad Gateway

检查 Go 服务是否正常运行：

```bash
curl http://localhost:8081/health
```

### 问题 3: URL 未替换

查看 Go 服务日志，确认 URL 替换逻辑正确执行。

### 查看日志

```bash
# Docker
docker logs clarity-proxy -f

# systemd
sudo journalctl -u clarity-proxy -f
```

## 性能

- **内存占用**: ~5-10 MB
- **CPU 占用**: < 1%
- **并发能力**: 数千并发请求
- **延迟**: < 10ms

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 相关链接

- [Microsoft Clarity 官网](https://clarity.microsoft.com/)
- [GitHub Container Registry](https://github.com/features/packages)
- [Docker Hub](https://hub.docker.com/)

---

**注意**: 请确保你有权使用 Microsoft Clarity 服务，并遵守其服务条款。此代理仅用于解决技术问题，不得用于任何违反服务条款的用途。
