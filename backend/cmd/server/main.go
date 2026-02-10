package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/andres10976/SISAP-PoC/backend/internal/config"
	"github.com/andres10976/SISAP-PoC/backend/internal/database"
	"github.com/andres10976/SISAP-PoC/backend/internal/handler"
	"github.com/andres10976/SISAP-PoC/backend/internal/middleware"
	"github.com/andres10976/SISAP-PoC/backend/internal/repository"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/monitor"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	// Database
	pool, err := database.Connect(cfg.DatabaseURL)
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

	// Services
	ctClient := ctlog.NewClient(cfg.CTLogURL)
	mon := monitor.New(ctClient, keywordRepo, certRepo, monitorRepo, cfg.MonitorBatchSize, cfg.MonitorInterval)

	// Handlers
	kwHandler := handler.NewKeywordHandler(keywordRepo)
	certHandler := handler.NewCertificateHandler(certRepo)
	monHandler := handler.NewMonitorHandler(mon, monitorRepo)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.CORS(cfg.CORSAllowOrigin))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	r.Route("/api/v1", func(r chi.Router) {
		kwHandler.RegisterRoutes(r)
		certHandler.RegisterRoutes(r)
		monHandler.RegisterRoutes(r)
	})

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
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
