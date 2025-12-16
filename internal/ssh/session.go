package ssh

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// 默认缓冲区大小
const defaultBufferSize = 32 * 1024

// SessionConfig SSH 会话配置
type SessionConfig struct {
	Hostname   string
	Port       int
	Username   string
	Password   string
	PrivateKey string
	Passphrase string
	Timeout    time.Duration
	BufferSize int
}

// Session SSH 会话
type Session struct {
	ID         string
	client     *ssh.Client
	session    *ssh.Session
	stdin      io.WriteCloser
	stdout     io.Reader
	stderr     io.Reader
	closed     bool
	closeMux   sync.Mutex
	bufferSize int
}

// NewSession 创建新的 SSH 会话
func NewSession(cfg SessionConfig) (*Session, error) {
	// 设置缓冲区大小
	bufSize := cfg.BufferSize
	if bufSize <= 0 {
		bufSize = defaultBufferSize
	}

	// 构建认证方法
	var authMethods []ssh.AuthMethod

	// 私钥认证
	if cfg.PrivateKey != "" {
		var signer ssh.Signer
		var err error

		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(cfg.PrivateKey), []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		}

		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 密码认证
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
		// 键盘交互认证（用于某些服务器）
		authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = cfg.Password
			}
			return answers, nil
		}))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("需要提供密码或私钥")
	}

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: 生产环境应验证主机密钥
		Timeout:         cfg.Timeout,
	}

	// 连接 SSH 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Hostname, cfg.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}

	return &Session{
		ID:         generateSessionID(),
		client:     client,
		bufferSize: bufSize,
	}, nil
}

// StartShell 启动 shell
func (s *Session) StartShell() error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("创建会话失败: %v", err)
	}
	s.session = session

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// 请求伪终端
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		return fmt.Errorf("请求 PTY 失败: %v", err)
	}

	// 获取 stdin、stdout、stderr
	s.stdin, err = session.StdinPipe()
	if err != nil {
		return fmt.Errorf("获取 stdin 失败: %v", err)
	}

	s.stdout, err = session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取 stdout 失败: %v", err)
	}

	s.stderr, err = session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取 stderr 失败: %v", err)
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("启动 shell 失败: %v", err)
	}

	return nil
}

// ResizeMessage 终端大小调整消息
type ResizeMessage struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// InputMessage WebSocket 输入消息
type InputMessage struct {
	Type   string        `json:"type"`
	Data   string        `json:"data,omitempty"`
	Resize ResizeMessage `json:"resize,omitempty"`
}

// HandleWebSocket 处理 WebSocket 连接
func (s *Session) HandleWebSocket(conn *websocket.Conn) {
	defer s.Close()

	// 用于通知 goroutine 退出
	done := make(chan struct{})
	defer close(done)

	// 从 SSH 读取数据并发送到 WebSocket
	go func() {
		buf := make([]byte, s.bufferSize)
		for {
			select {
			case <-done:
				return
			default:
				n, err := s.stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("读取 stdout 错误: %v", err)
					}
					conn.Close()
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("发送 WebSocket 消息错误: %v", err)
						return
					}
				}
			}
		}
	}()

	// 同时读取 stderr
	go func() {
		buf := make([]byte, s.bufferSize)
		for {
			select {
			case <-done:
				return
			default:
				n, err := s.stderr.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					conn.WriteMessage(websocket.BinaryMessage, buf[:n])
				}
			}
		}
	}()

	// 从 WebSocket 读取数据并发送到 SSH
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 关闭: %v", err)
			}
			return
		}

		if msgType == websocket.TextMessage {
			// 尝试解析为 JSON 消息
			var input InputMessage
			if err := json.Unmarshal(msg, &input); err == nil {
				switch input.Type {
				case "resize":
					if s.session != nil {
						s.session.WindowChange(input.Resize.Rows, input.Resize.Cols)
					}
					continue
				case "data":
					msg = []byte(input.Data)
				}
			}
		}

		// 写入到 SSH stdin
		if _, err := s.stdin.Write(msg); err != nil {
			log.Printf("写入 SSH 错误: %v", err)
			return
		}
	}
}

// Close 关闭会话
func (s *Session) Close() {
	s.closeMux.Lock()
	defer s.closeMux.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
	log.Printf("会话已关闭: %s", s.ID)
}

// generateSessionID 生成会话 ID
func generateSessionID() string {
	return uuid.New().String()
}
