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
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"

	gateway "github.com/mainframe/tm-system/gateway/internal"
)

// GatewayConfig extends BaseConfig with HTTP-specific settings.
type GatewayConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	HTTP              HTTPConfig `mapstructure:"http"`
	IngestSimURL      string     `mapstructure:"ingest_sim_url"` // default "http://localhost:8082"
}

// HTTPConfig holds the HTTP server port.
type HTTPConfig struct {
	Port int `mapstructure:"port"`
}

func main() {
	// --- Load configuration ---------------------------------------------------
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	var cfg GatewayConfig
	if err := config.Load(configPath, &cfg); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// --- Logger ---------------------------------------------------------------
	level := logging.ParseLevel(cfg.Service.LogLevel)
	logger := logging.NewLogger(cfg.Service.Name, level)

	// --- Graceful shutdown context --------------------------------------------
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Redis client ---------------------------------------------------------
	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// --- SQLite client --------------------------------------------------------
	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		logger.Error("failed to open SQLite", "error", err)
		os.Exit(1)
	}
	defer sdb.Close()

	// --- Chi router with CORS -------------------------------------------------
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	// --- Register routes ------------------------------------------------------
	if cfg.IngestSimURL == "" {
		cfg.IngestSimURL = "http://localhost:8082"
	}
	gateway.RegisterRoutes(r, rdb, sdb, cfg.IngestSimURL, logger)

	// --- Start HTTP server ----------------------------------------------------
	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info("starting HTTP server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("server stopped gracefully")
}

// corsMiddleware adds CORS headers allowing all origins with specified methods and headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
