# 多阶段构建 Dockerfile
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 复制 go mod 文件（如果有）
COPY go.* ./
RUN if [ -f go.mod ]; then go mod download; fi

# 复制源代码
COPY clarity-proxy.go .

# 编译静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o clarity-proxy clarity-proxy.go

# 最终镜像
FROM alpine:latest

# 安装 CA 证书（用于 HTTPS 请求）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 clarity && \
    adduser -D -u 1000 -G clarity clarity

# 设置工作目录
WORKDIR /app

# 从 builder 阶段复制二进制文件
COPY --from=builder /build/clarity-proxy .

# 修改文件所有者
RUN chown -R clarity:clarity /app

# 切换到非 root 用户
USER clarity

# 暴露端口
EXPOSE 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 启动服务
ENTRYPOINT ["/app/clarity-proxy"]
