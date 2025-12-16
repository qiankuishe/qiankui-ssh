package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiter IP 速率限制器
type RateLimiter struct {
	requests map[string]*requestInfo
	mu       sync.RWMutex
	limit    int           // 最大请求数
	window   time.Duration // 时间窗口
}

type requestInfo struct {
	count     int
	firstTime time.Time
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*requestInfo),
		limit:    limit,
		window:   window,
	}
	// 启动清理协程
	go rl.cleanup()
	return rl
}

// cleanup 定期清理过期记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, info := range rl.requests {
			if now.Sub(info.firstTime) > rl.window {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Check 检查是否允许请求
func (rl *RateLimiter) Check(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = &requestInfo{
			count:     1,
			firstTime: now,
		}
		return true
	}

	// 如果时间窗口已过，重置计数
	if now.Sub(info.firstTime) > rl.window {
		info.count = 1
		info.firstTime = now
		return true
	}

	// 检查是否超过限制
	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

// Middleware 返回 Fiber 中间件
func (rl *RateLimiter) Middleware(paths ...string) fiber.Handler {
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	return func(c *fiber.Ctx) error {
		// 只对指定路径限流
		if len(pathSet) > 0 && !pathSet[c.Path()] {
			return c.Next()
		}

		ip := c.IP()
		if !rl.Check(ip) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "请求过于频繁，请稍后再试",
			})
		}
		return c.Next()
	}
}

// SecurityHeaders 安全头部中间件
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Content Security Policy
		c.Set("Content-Security-Policy", 
			"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; "+
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; "+
			"font-src 'self' https://cdnjs.cloudflare.com; "+
			"img-src 'self' data:; "+
			"connect-src 'self' ws: wss:; "+
			"frame-ancestors 'none'")
		
		// 其他安全头部
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		return c.Next()
	}
}
