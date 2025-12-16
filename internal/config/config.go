package config

import (
	"flag"
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	// 服务器配置
	Address string
	Port    int

	// SSH 配置
	Timeout    int
	MaxConn    int
	BufferSize int

	// 安全配置
	AllowOrigins string

	// 其他
	Debug   bool
	Version bool
}

// ParseFlags 解析命令行参数
func ParseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "address", getEnv("QKSSH_ADDRESS", "0.0.0.0"), "监听地址")
	flag.IntVar(&cfg.Port, "port", getEnvInt("QKSSH_PORT", 8888), "监听端口")
	flag.IntVar(&cfg.Timeout, "timeout", getEnvInt("QKSSH_TIMEOUT", 10), "SSH 连接超时(秒)")
	flag.IntVar(&cfg.MaxConn, "maxconn", getEnvInt("QKSSH_MAXCONN", 100), "最大连接数")
	flag.IntVar(&cfg.BufferSize, "buffer", getEnvInt("QKSSH_BUFFER", 32768), "缓冲区大小")
	flag.StringVar(&cfg.AllowOrigins, "origins", getEnv("QKSSH_ORIGINS", "*"), "允许的来源")
	flag.BoolVar(&cfg.Debug, "debug", getEnvBool("QKSSH_DEBUG", false), "调试模式")
	flag.BoolVar(&cfg.Version, "version", false, "显示版本信息")

	flag.Parse()

	return cfg
}

// getEnv 获取环境变量，带默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
