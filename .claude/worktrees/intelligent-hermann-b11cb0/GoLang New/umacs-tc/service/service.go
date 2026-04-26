package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mainframe/umacs-tc/internal"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

// Config holds the umacs-tc configuration.
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Umacs struct {
		TcIP                string `yaml:"tc_ip"`
		TcPort              string `yaml:"tc_port"`
		DataServerIP        string `yaml:"data_server_ip"`
		APIReqSource        string `yaml:"api_req_source"`
		APIReqPriority      string `yaml:"api_req_priority"`
		APIReqExecutionMode string `yaml:"api_req_execution_mode"`
		APIReqSubsystem     string `yaml:"api_req_subsystem"`
	} `yaml:"umacs"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Run starts the umacs-tc service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("umacs-tc: failed to load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("umacs-tc: failed to connect to Redis: %w", err)
	}
	logger.Info("connected to redis")

	umacsEnv := internal.NewUmacsEnvData(rdb, &cfg.Umacs, logger)
	handler := internal.NewHandler(rdb, umacsEnv, logger)

	// Consumer context derived from the passed-in ctx so it can be cancelled
	// before HTTP server shutdown to stop processing new commands.
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	consumer := internal.NewQueueConsumer(rdb, handler, logger)
	go consumer.Run(consumerCtx)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		logger.Info("starting server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()

	// Stop consumer first, then drain HTTP.
	consumerCancel()
	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("umacs-tc: server shutdown error: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
