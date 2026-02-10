package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/andres10976/SISAP-PoC/backend/internal/database"
	"github.com/andres10976/SISAP-PoC/backend/internal/handler"
	"github.com/andres10976/SISAP-PoC/backend/internal/middleware"
	"github.com/andres10976/SISAP-PoC/backend/internal/repository"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/monitor"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Config
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}
	serverPort := getEnv("SERVER_PORT", "8080")
	ctLogURL := getEnv("CT_LOG_URL", "https://oak.ct.letsencrypt.org/2026h2")
	corsOrigin := getEnv("CORS_ALLOW_ORIGIN", "http://localhost:3000")
	monitorInterval := getDuration("MONITOR_INTERVAL", 60*time.Second)
	monitorBatchSize := getInt("MONITOR_BATCH_SIZE", 100)
	monitorReprocessOnIdle := getBool("MONITOR_REPROCESS_ON_IDLE", false)

	// Database
	pool, err := database.Connect(databaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(pool); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	// Repositories
	keywordRepo := repository.NewKeywordRepository(pool)
	certRepo := repository.NewCertificateRepository(pool)
	monitorRepo := repository.NewMonitorRepository(pool)

	// Reset stale monitor state from previous process crash
	if err := monitorRepo.SetRunning(context.Background(), false); err != nil {
		slog.Error("failed to reset monitor state", "error", err)
		os.Exit(1)
	}

	// Services
	ctClient := ctlog.NewClient(ctLogURL)
	mon := monitor.New(ctClient, keywordRepo, certRepo, monitorRepo, monitorBatchSize, monitorInterval, monitorReprocessOnIdle)

	// Handlers
	kwHandler := handler.NewKeywordHandler(keywordRepo)
	certHandler := handler.NewCertificateHandler(certRepo)
	monHandler := handler.NewMonitorHandler(mon, monitorRepo)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.CORS(corsOrigin))
	r.Use(chiMiddleware.Logger)
	r.Use(middleware.Recovery)

	r.Route("/api/v1", func(r chi.Router) {
		kwHandler.RegisterRoutes(r)
		certHandler.RegisterRoutes(r)
		monHandler.RegisterRoutes(r)
	})

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", serverPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	// Stop the monitor if running
	mon.Stop(context.Background())

	// Give in-flight requests time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
