package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"qiankui-ssh/internal/config"
	"qiankui-ssh/internal/handler"
	"qiankui-ssh/internal/middleware"
	"qiankui-ssh/web"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// 解析命令行参数
	cfg := config.ParseFlags()

	// 显示版本信息
	if cfg.Version {
		fmt.Println("Qiankui SSH v1.0.0")
		fmt.Println("千葵SSH - 轻量、快速、优雅的 Web SSH 终端")
		os.Exit(0)
	}

	// 初始化模板引擎
	engine := html.NewFileSystem(web.TemplateFS(), ".html")
	engine.Reload(cfg.Debug)

	// 创建 Fiber 应用
	app := fiber.New(fiber.Config{
		AppName:               "Qiankui SSH",
		Views:                 engine,
		DisableStartupMessage: !cfg.Debug,
	})

	// 基础中间件
	app.Use(recover.New())
	if cfg.Debug {
		app.Use(logger.New())
	}

	// 安全头部中间件
	app.Use(middleware.SecurityHeaders())

	// CORS 中间件
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// 速率限制 - 对 /connect 端点限制每分钟 10 次
	rateLimiter := middleware.NewRateLimiter(10, time.Minute)
	app.Use("/connect", rateLimiter.Middleware("/connect"))

	// 静态文件服务
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   web.StaticFS(),
		Browse: false,
	}))

	// 路由
	h := handler.New(cfg)

	// 首页
	app.Get("/", h.Index)

	// 健康检查
	app.Get("/health", h.Health)

	// SSH 连接接口
	app.Post("/connect", h.Connect)

	// WebSocket 终端
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(h.WebSocket))

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	log.Printf("Qiankui SSH 正在启动...")
	log.Printf("监听地址: http://%s", addr)
	log.Printf("调试模式: %v", cfg.Debug)

	if err := app.Listen(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
