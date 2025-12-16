English | [中文](./README.md)

# Qiankui SSH

**Qiankui SSH** - A lightweight, fast, and elegant Web SSH terminal

<p align="center">
  <img src="web/static/img/favicon.svg" width="120" alt="Qiankui SSH Logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker" alt="Docker">
</p>

---

## Features

- **Single Binary** - Just one executable file after compilation
- **High Performance** - Native Go concurrency handles thousands of connections
- **Tiny Image** - Docker image is only ~15MB
- **Multiple Auth** - Supports password and private key authentication
- **Security Hardened** - Rate limiting, CSP headers, connection limits

---

## Quick Start

### Docker One-liner

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
      - QKSSH_ORIGINS=https://your-domain.com  # Set specific domain in production
    restart: unless-stopped
```

### Build from Source

```bash
git clone https://github.com/qiankuishe/qiankui-ssh.git
cd qiankui-ssh
go mod tidy
go build -o qkssh ./cmd/qkssh
./qkssh --port=8888
```

---

## Command Line Options

| Flag | Environment | Default | Description |
|------|-------------|---------|-------------|
| `--address` | `QKSSH_ADDRESS` | `0.0.0.0` | Listen address |
| `--port` | `QKSSH_PORT` | `8888` | Listen port |
| `--timeout` | `QKSSH_TIMEOUT` | `10` | SSH connection timeout (seconds) |
| `--maxconn` | `QKSSH_MAXCONN` | `100` | Maximum concurrent connections |
| `--origins` | `QKSSH_ORIGINS` | `*` | Allowed CORS origins |
| `--debug` | `QKSSH_DEBUG` | `false` | Debug mode |

---

## Security Measures

| Measure | Description |
|---------|-------------|
| Rate Limiting | `/connect` endpoint: 10 requests per IP per minute |
| Session Cleanup | Unused sessions are cleaned up after 30 seconds |
| Connection Limit | Maximum 100 concurrent connections by default |
| CSP Headers | Prevents XSS and resource injection attacks |
| Security Headers | X-Frame-Options, X-Content-Type-Options, etc. |
| Non-root Container | Docker runs as unprivileged user |

---

## Important Notes

### Deployment Recommendations

1. **Use HTTPS in Production**
   - Use Nginx/Caddy as reverse proxy with TLS
   - Otherwise passwords are transmitted in plaintext

2. **Configure CORS Whitelist**
   ```bash
   ./qkssh --origins="https://your-domain.com"
   ```

3. **Restrict Listen Address**
   - For internal use only, bind to internal IP
   ```bash
   ./qkssh --address=192.168.1.100
   ```

### Known Limitations

- **SSH Host Key Not Verified** - MITM attack risk, use only in trusted networks
- **Password in URL Parameters** - Though Base64 encoded, may be logged

---

## Security Concerns

> **Important**: This tool acts as an SSH proxy and handles sensitive credentials

### High-Risk Scenarios

| Scenario | Risk | Recommendation |
|----------|------|----------------|
| Public Exposure | Brute force, credential theft | Use with VPN or jump server |
| Multi-user Sharing | Credential leakage | Deploy separate instances |
| Logging | Connection info may be logged | Regularly clean logs |

### NOT Recommended For

- Primary management interface for production servers
- Servers storing sensitive data
- Compliance environments requiring audit trails

### Recommended Use Cases

- Internal operations tools
- Temporary remote access
- Development/testing environments
- Home server management

---

## Project Structure

```
qiankui-ssh/
├── cmd/qkssh/main.go           # Entry point
├── internal/
│   ├── config/config.go        # Configuration
│   ├── handler/handler.go      # HTTP/WebSocket handlers
│   ├── middleware/security.go  # Security middleware
│   └── ssh/session.go          # SSH session management
├── web/
│   ├── embed.go                # Embedded files
│   ├── static/                 # Static assets
│   └── templates/              # HTML templates
├── Dockerfile                  # Multi-stage build
└── docker-compose.yml
```

---

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Main page |
| `/health` | GET | Health check |
| `/connect` | POST | Establish SSH connection (rate limited) |
| `/ws` | WebSocket | Terminal data transmission |

---

## Tech Stack

- **Backend**: Go 1.21+, Fiber v2
- **Frontend**: xterm.js 5.3.0, Vanilla CSS
- **Deployment**: Docker, GitHub Actions (multi-arch)

---

## License

MIT License

---

## Credits

- [WebSSH](https://github.com/huashengdun/webssh) - Inspiration
- [Fiber](https://gofiber.io/) - Web framework
- [xterm.js](https://xtermjs.org/) - Terminal emulator
