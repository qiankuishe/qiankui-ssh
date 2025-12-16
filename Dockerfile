# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o qkssh ./cmd/qkssh

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装 CA 证书和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 qkssh && \
    adduser -u 1000 -G qkssh -s /bin/sh -D qkssh

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制文件
COPY --from=builder /app/qkssh .

# 设置文件权限
RUN chown qkssh:qkssh /app/qkssh

# 切换到非 root 用户
USER qkssh

# 暴露端口
EXPOSE 8888

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/health || exit 1

# 运行
ENTRYPOINT ["./qkssh"]
CMD ["--address=0.0.0.0", "--port=8888"]
