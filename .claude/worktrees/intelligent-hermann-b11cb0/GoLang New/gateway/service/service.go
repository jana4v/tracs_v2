package service

import (
	"context"
	"fmt"
	"net/http"
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

// Run starts the gateway HTTP server and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg GatewayConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("gateway: config error: %w", err)
	}

	level := logging.ParseLevel(cfg.Service.LogLevel)
	logger := logging.NewLogger(cfg.Service.Name, level)

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("gateway: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("gateway: failed to open SQLite: %w", err)
	}
	defer sdb.Close()

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	if cfg.IngestSimURL == "" {
		cfg.IngestSimURL = "http://localhost:8082"
	}
	gateway.RegisterRoutes(r, rdb, sdb, cfg.IngestSimURL, logger)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting HTTP server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("gateway: HTTP server shutdown error: %w", err)
	}

	logger.Info("server stopped gracefully")
	return nil
}

// corsMiddleware adds CORS headers allowing all origins with specified methods and headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
