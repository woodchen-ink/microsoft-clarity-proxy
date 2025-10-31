# 快速开始指南

5 分钟内完成 Microsoft Clarity Proxy 的部署和使用。

## 前提条件

- 一台服务器（Linux/Windows）
- Docker 已安装
- 一个域名（如 `analytics.example.com`）
- SSL 证书（可使用 Let's Encrypt）

## 步骤 1: 启动 Go 代理服务

```bash
docker run -d \
  --name clarity-proxy \
  --restart unless-stopped \
  -p 127.0.0.1:8081:8081 \
  -e PROXY_DOMAIN=https://analytics.example.com \
  ghcr.io/YOUR_USERNAME/microsoft-clarity:latest
```

验证服务运行：

```bash
curl http://localhost:8081/health
# 应该返回: {"status":"ok","service":"clarity-proxy","domain":"https://analytics.example.com"}
```

## 步骤 2: 配置 Nginx

创建 nginx 配置文件 `/etc/nginx/sites-available/analytics.example.com`:

```nginx
server {
    listen 443 ssl http2;
    server_name analytics.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /ms/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Origin $http_origin;

        # 禁用缓冲和压缩,避免 HTTP/2 协议错误
        proxy_buffering off;
        proxy_request_buffering off;
        gzip off;
    }
}
```

启用站点并重载 nginx：

```bash
sudo ln -s /etc/nginx/sites-available/analytics.example.com \
           /etc/nginx/sites-enabled/

sudo nginx -t && sudo nginx -s reload
```

## 步骤 3: 测试配置

```bash
# 测试主脚本
curl "https://analytics.example.com/ms/t.js?id=test123"

# 测试健康检查
curl "https://analytics.example.com/health"

# 验证 CORS 头
curl -H "Origin: https://yourwebsite.com" \
     -I https://analytics.example.com/ms/p
```

应该看到：
```
access-control-allow-origin: https://yourwebsite.com
access-control-allow-credentials: true
```

## 步骤 4: 集成到网站

在你的网站 HTML 中添加：

```html
<script type="text/javascript">
    (function(c,l,a,r,i,t,y){
        c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};
        t=l.createElement(r);t.async=1;
        t.src="https://analytics.example.com/ms/t.js?id=YOUR_CLARITY_PROJECT_ID";
        y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y);
    })(window, document, "clarity", "script", "YOUR_CLARITY_PROJECT_ID");
</script>
```

替换：
- `analytics.example.com` → 你的代理域名
- `YOUR_CLARITY_PROJECT_ID` → 你的 Clarity 项目 ID（在 Clarity 控制台获取）

## 步骤 5: 验证数据收集

1. 访问你的网站
2. 打开浏览器开发者工具 → Network 标签
3. 查看请求：
   - ✅ `https://analytics.example.com/ms/t.js?id=...` → 200 OK
   - ✅ `https://analytics.example.com/ms/j/...` → 200 OK
   - ✅ `https://analytics.example.com/ms/p` → 200 OK
4. 登录 [Clarity 控制台](https://clarity.microsoft.com/) 查看数据

## 完成！🎉

你的 Clarity 代理现在已经运行了！

## 下一步

- [查看完整文档](README.md)
- [部署指南](DEPLOY.md)
- [故障排查](README.md#故障排查)

## 常见问题

### Q: 如何获取 Clarity 项目 ID？

1. 访问 https://clarity.microsoft.com/
2. 登录并创建项目
3. 在项目设置中找到项目 ID（格式：`abc123defg`）

### Q: 如何验证代理是否正常工作？

```bash
# 检查服务状态
docker ps | grep clarity-proxy

# 查看日志
docker logs clarity-proxy

# 测试端点
curl http://localhost:8081/health
```

### Q: 如何更新到最新版本？

```bash
docker pull ghcr.io/YOUR_USERNAME/microsoft-clarity:latest
docker stop clarity-proxy
docker rm clarity-proxy
# 然后重新运行步骤 1 的命令
```

### Q: 遇到 CORS 错误怎么办？

确保：
1. `PROXY_DOMAIN` 环境变量正确设置
2. 前端代码中的域名与 `PROXY_DOMAIN` 一致
3. nginx 正确转发了所有请求头

### Q: 如何查看实时日志？

```bash
docker logs -f clarity-proxy
```

## 需要帮助？

- 查看 [README](README.md)
- 查看 [部署文档](DEPLOY.md)
- 提交 [GitHub Issue](https://github.com/YOUR_USERNAME/microsoft-clarity/issues)
