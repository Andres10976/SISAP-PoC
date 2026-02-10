package config

import (
	"os"
	"testing"
	"time"
)

func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg := Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.CTLogURL != "https://oak.ct.letsencrypt.org/2026h2" {
		t.Errorf("CTLogURL = %q, want default", cfg.CTLogURL)
	}
	if cfg.MonitorInterval != 60*time.Second {
		t.Errorf("MonitorInterval = %v, want 60s", cfg.MonitorInterval)
	}
	if cfg.MonitorBatchSize != 100 {
		t.Errorf("MonitorBatchSize = %d, want 100", cfg.MonitorBatchSize)
	}
	if cfg.CORSAllowOrigin != "http://localhost:3000" {
		t.Errorf("CORSAllowOrigin = %q, want default", cfg.CORSAllowOrigin)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	setEnvs(t, map[string]string{
		"DATABASE_URL":      "postgres://custom/db",
		"SERVER_PORT":       "9090",
		"CT_LOG_URL":        "https://custom.ct.log",
		"MONITOR_INTERVAL":  "30s",
		"MONITOR_BATCH_SIZE": "250",
		"CORS_ALLOW_ORIGIN": "https://example.com",
	})

	cfg := Load()

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
	}
	if cfg.DatabaseURL != "postgres://custom/db" {
		t.Errorf("DatabaseURL = %q, want custom", cfg.DatabaseURL)
	}
	if cfg.CTLogURL != "https://custom.ct.log" {
		t.Errorf("CTLogURL = %q, want custom", cfg.CTLogURL)
	}
	if cfg.MonitorInterval != 30*time.Second {
		t.Errorf("MonitorInterval = %v, want 30s", cfg.MonitorInterval)
	}
	if cfg.MonitorBatchSize != 250 {
		t.Errorf("MonitorBatchSize = %d, want 250", cfg.MonitorBatchSize)
	}
	if cfg.CORSAllowOrigin != "https://example.com" {
		t.Errorf("CORSAllowOrigin = %q, want custom", cfg.CORSAllowOrigin)
	}
}

func TestLoad_MissingDatabaseURL_Panics(t *testing.T) {
	os.Unsetenv("DATABASE_URL")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for missing DATABASE_URL")
		}
		msg, ok := r.(string)
		if !ok || msg == "" {
			t.Errorf("expected string panic message, got %v", r)
		}
	}()

	Load()
}

func TestLoad_InvalidDuration_FallsBack(t *testing.T) {
	setEnvs(t, map[string]string{
		"DATABASE_URL":     "postgres://localhost/test",
		"MONITOR_INTERVAL": "not-a-duration",
	})

	cfg := Load()

	if cfg.MonitorInterval != 60*time.Second {
		t.Errorf("MonitorInterval = %v, want fallback 60s", cfg.MonitorInterval)
	}
}

func TestLoad_InvalidInt_FallsBack(t *testing.T) {
	setEnvs(t, map[string]string{
		"DATABASE_URL":       "postgres://localhost/test",
		"MONITOR_BATCH_SIZE": "abc",
	})

	cfg := Load()

	if cfg.MonitorBatchSize != 100 {
		t.Errorf("MonitorBatchSize = %d, want fallback 100", cfg.MonitorBatchSize)
	}
}
