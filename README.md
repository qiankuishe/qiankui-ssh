[English](./README_EN.md) | 中文

# Qiankui SSH

**千葵SSH** - 轻量、快速、优雅的 Web SSH 终端

<p align="center">
  <img src="web/static/img/favicon.svg" width="120" alt="Qiankui SSH Logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker" alt="Docker">
</p>

---

## 特性

- **单文件部署** - 编译后仅一个可执行文件
- **高性能** - Go 原生并发，轻松处理数千连接
- **超小镜像** - Docker 镜像仅 ~15MB
- **多种认证** - 支持密码、私钥认证
- **安全加固** - 速率限制、CSP 头部、连接数限制

---

## 快速开始

### Docker 一键部署

```bash
docker run -d --name qiankui-ssh -p 8888:8888 ghcr.io/qiankuishe/qiankui-ssh:main
```

### Docker Compose

```yaml
version: '3.8'
services:
  qiankui-ssh:
    image: ghcr.io/qiankuishe/qiankui-ssh:main
    container_name: qiankui-ssh
    ports:
      - "8888:8888"
    environment:
      - QKSSH_ORIGINS=https://your-domain.com  # 生产环境设置具体域名
    restart: unless-stopped
```

### 手动编译

```bash
git clone https://github.com/qiankuishe/qiankui-ssh.git
cd qiankui-ssh
go mod tidy
go build -o qkssh ./cmd/qkssh
./qkssh --port=8888
```

---

## 命令行参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `--address` | `QKSSH_ADDRESS` | `0.0.0.0` | 监听地址 |
| `--port` | `QKSSH_PORT` | `8888` | 监听端口 |
| `--timeout` | `QKSSH_TIMEOUT` | `10` | SSH 连接超时(秒) |
| `--maxconn` | `QKSSH_MAXCONN` | `100` | 最大并发连接数 |
| `--origins` | `QKSSH_ORIGINS` | `*` | 允许的 CORS 来源 |
| `--debug` | `QKSSH_DEBUG` | `false` | 调试模式 |

---

## 安全措施

| 措施 | 说明 |
|------|------|
| 速率限制 | `/connect` 端点每 IP 每分钟最多 10 次请求 |
| 会话清理 | 30 秒未使用的会话自动清理 |
| 连接数限制 | 默认最多 100 个并发连接 |
| CSP 头部 | 防止 XSS 和资源注入攻击 |
| 安全头部 | X-Frame-Options, X-Content-Type-Options 等 |
| 非 root 容器 | Docker 以普通用户运行 |

---

## 注意事项

### 部署建议

1. **生产环境必须使用 HTTPS**
   - 配合 Nginx/Caddy 反向代理添加 TLS
   - 否则密码将以明文在网络上传输

2. **设置 CORS 白名单**
   ```bash
   ./qkssh --origins="https://your-domain.com"
   ```

3. **限制监听地址**
   - 如仅内网使用，绑定到内网 IP
   ```bash
   ./qkssh --address=192.168.1.100
   ```

### 已知限制

- **SSH 主机密钥未验证** - 存在中间人攻击风险，仅建议在可信网络使用
- **URL 参数传递密码** - 虽然 Base64 编码，但仍可能被日志记录

---

## 安全性担忧

> **重要提示**：此工具作为 SSH 代理，会接触用户的敏感凭据

### 高风险场景

| 场景 | 风险 | 建议 |
|------|------|------|
| 公网暴露 | 暴力破解、凭据泄露 | 配合 VPN 或跳板机使用 |
| 多人共用 | 凭据被其他用户窃取 | 每人独立部署实例 |
| 日志记录 | 连接信息可能被记录 | 定期清理日志 |

### 不建议用于

- 生产服务器的主要管理入口
- 存储敏感数据的服务器
- 需要审计追踪的合规环境

### 建议用途

- 内网运维工具
- 临时远程访问
- 开发测试环境
- 家庭服务器管理

---

## 项目结构

```
qiankui-ssh/
├── cmd/qkssh/main.go           # 程序入口
├── internal/
│   ├── config/config.go        # 配置管理
│   ├── handler/handler.go      # HTTP/WebSocket 处理
│   ├── middleware/security.go  # 安全中间件
│   └── ssh/session.go          # SSH 会话管理
├── web/
│   ├── embed.go                # 嵌入文件
│   ├── static/                 # 静态资源
│   └── templates/              # HTML 模板
├── Dockerfile                  # 多阶段构建
└── docker-compose.yml
```

---

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 主页面 |
| `/health` | GET | 健康检查 |
| `/connect` | POST | 建立 SSH 连接 (有速率限制) |
| `/ws` | WebSocket | 终端数据传输 |

---

## 技术栈

- **后端**: Go 1.21+, Fiber v2
- **前端**: xterm.js 5.3.0, Vanilla CSS
- **部署**: Docker, GitHub Actions (多架构)

---

## 许可证

MIT License

---

## 致谢

- [WebSSH](https://github.com/huashengdun/webssh) - 项目灵感来源
- [Fiber](https://gofiber.io/) - Web 框架
- [xterm.js](https://xtermjs.org/) - 终端模拟器
