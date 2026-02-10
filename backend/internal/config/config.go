package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	CTLogURL         string
	MonitorInterval  time.Duration
	MonitorBatchSize int
	CORSAllowOrigin  string
}

func Load() *Config {
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		DatabaseURL:      requireEnv("DATABASE_URL"),
		CTLogURL:         getEnv("CT_LOG_URL", "https://oak.ct.letsencrypt.org/2026h2"),
		MonitorInterval:  getDuration("MONITOR_INTERVAL", 60*time.Second),
		MonitorBatchSize: getInt("MONITOR_BATCH_SIZE", 100),
		CORSAllowOrigin:  getEnv("CORS_ALLOW_ORIGIN", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration for env var, using default",
			"key", key, "value", v, "default", fallback)
		return fallback
	}
	return d
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("invalid integer for env var, using default",
			"key", key, "value", v, "default", fallback)
		return fallback
	}
	return n
}
