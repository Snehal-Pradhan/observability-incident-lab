package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort        int
	DatabaseURL     string
	RedisAddr       string
	DBMaxConns      int32
	CacheTTLSeconds int
	LogLevel        string
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	cfg := Config{
		HTTPPort:        8080,
		DatabaseURL:     databaseURL,
		RedisAddr:       "localhost:6379",
		DBMaxConns:      10,
		CacheTTLSeconds: 300,
		LogLevel:        "info",
	}

	if value := os.Getenv("HTTP_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
		}
		cfg.HTTPPort = port
	}

	if value := os.Getenv("REDIS_ADDR"); value != "" {
		cfg.RedisAddr = value
	}

	if value := os.Getenv("DB_MAX_CONNS"); value != "" {
		maxConns, err := strconv.ParseInt(value, 10, 32)
		if err != nil || maxConns < 1 {
			return Config{}, fmt.Errorf("DB_MAX_CONNS must be a positive integer")
		}
		cfg.DBMaxConns = int32(maxConns)
	}

	if value := os.Getenv("CACHE_TTL_SECONDS"); value != "" {
		ttl, err := strconv.Atoi(value)
		if err != nil || ttl < 0 {
			return Config{}, fmt.Errorf("CACHE_TTL_SECONDS must be a non-negative integer")
		}
		cfg.CacheTTLSeconds = ttl
	}

	if value := os.Getenv("LOG_LEVEL"); value != "" {
		level := strings.ToLower(strings.TrimSpace(value))
		switch level {
		case "debug", "info", "warn", "error":
			cfg.LogLevel = level
		default:
			return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
		}
	}

	return cfg, nil
}
