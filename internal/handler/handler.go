package handler

import (
	"encoding/base64"
	"log"
	"sync"
	"time"

	"qiankui-ssh/internal/config"
	"qiankui-ssh/internal/ssh"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Handler HTTP 请求处理器
type Handler struct {
	cfg      *config.Config
	sessions sync.Map // sessionID -> *sessionEntry
}

// sessionEntry 会话条目，包含创建时间
type sessionEntry struct {
	session   *ssh.Session
	createdAt time.Time
}

// New 创建新的 Handler
func New(cfg *config.Config) *Handler {
	h := &Handler{
		cfg: cfg,
	}
	// 启动会话清理协程
	go h.startSessionCleaner()
	return h
}

// startSessionCleaner 定期清理过期会话
func (h *Handler) startSessionCleaner() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		h.sessions.Range(func(key, value interface{}) bool {
			entry := value.(*sessionEntry)
			// 会话创建后30秒内未使用则清理
			if now.Sub(entry.createdAt) > 30*time.Second {
				log.Printf("清理过期会话: %s", key)
				entry.session.Close()
				h.sessions.Delete(key)
			}
			return true
		})
	}
}

// Health 健康检查
func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": "1.0.0",
	})
}

// Index 首页
func (h *Handler) Index(c *fiber.Ctx) error {
	// 检查 URL 参数是否包含连接信息
	hostname := c.Query("hostname")
	port := c.Query("port", "22")
	username := c.Query("username", "root")
	password := c.Query("password")
	command := c.Query("command")

	// 如果密码是 base64 编码的，解码它
	if password != "" {
		if decoded, err := base64.StdEncoding.DecodeString(password); err == nil {
			password = string(decoded)
		}
	}

	return c.Render("index", fiber.Map{
		"Hostname": hostname,
		"Port":     port,
		"Username": username,
		"Password": password,
		"Command":  command,
	})
}

// ConnectRequest 连接请求
type ConnectRequest struct {
	Hostname   string `json:"hostname" form:"hostname"`
	Port       int    `json:"port" form:"port"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	PrivateKey string `json:"privatekey" form:"privatekey"`
	Passphrase string `json:"passphrase" form:"passphrase"`
}

// ConnectResponse 连接响应
type ConnectResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Connect 建立 SSH 连接
func (h *Handler) Connect(c *fiber.Ctx) error {
	var req ConnectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "请求参数错误",
		})
	}

	// 验证必填参数
	if req.Hostname == "" {
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "主机地址不能为空",
		})
	}
	if req.Username == "" {
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "用户名不能为空",
		})
	}
	
	// 验证端口范围
	if req.Port == 0 {
		req.Port = 22
	} else if req.Port < 1 || req.Port > 65535 {
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "端口范围无效 (1-65535)",
		})
	}

	// 检查当前连接数
	var connCount int
	h.sessions.Range(func(key, value interface{}) bool {
		connCount++
		return true
	})
	if connCount >= h.cfg.MaxConn {
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "连接数已达上限",
		})
	}

	// 创建 SSH 会话
	session, err := ssh.NewSession(ssh.SessionConfig{
		Hostname:   req.Hostname,
		Port:       req.Port,
		Username:   req.Username,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
		Timeout:    time.Duration(h.cfg.Timeout) * time.Second,
		BufferSize: h.cfg.BufferSize,
	})
	if err != nil {
		log.Printf("SSH 连接失败: %v", err)
		return c.JSON(ConnectResponse{
			Success: false,
			Message: "连接失败，请检查地址和凭据",
		})
	}

	// 保存会话
	h.sessions.Store(session.ID, &sessionEntry{
		session:   session,
		createdAt: time.Now(),
	})

	log.Printf("SSH 连接成功: %s@%s:%d [%s]", req.Username, req.Hostname, req.Port, session.ID)

	return c.JSON(ConnectResponse{
		Success:   true,
		SessionID: session.ID,
	})
}

// WebSocket 处理 WebSocket 连接
func (h *Handler) WebSocket(c *websocket.Conn) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.WriteMessage(websocket.TextMessage, []byte("错误: 缺少 session_id"))
		c.Close()
		return
	}

	// 获取会话
	value, ok := h.sessions.Load(sessionID)
	if !ok {
		c.WriteMessage(websocket.TextMessage, []byte("错误: 会话不存在或已过期"))
		c.Close()
		return
	}
	entry := value.(*sessionEntry)

	// 从会话映射中删除（防止重复使用）
	h.sessions.Delete(sessionID)

	// 启动终端
	if err := entry.session.StartShell(); err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("错误: 启动终端失败"))
		c.Close()
		entry.session.Close()
		return
	}

	// 处理终端数据
	entry.session.HandleWebSocket(c)
}
