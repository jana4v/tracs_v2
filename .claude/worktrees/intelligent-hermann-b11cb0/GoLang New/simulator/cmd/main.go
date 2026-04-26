package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"
	"github.com/mainframe/tm-system/internal/repository/sqlite"
	sim "github.com/mainframe/tm-system/simulator/internal"
)

// SimulatorConfig extends BaseConfig with HTTP port.
type SimulatorConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	HTTP              struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"http"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	var cfg SimulatorConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = 21001
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	// Context with graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Connect to Redis
	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Open SQLite
	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		logger.Error("failed to open SQLite", "error", err)
		os.Exit(1)
	}
	defer sdb.Close()

	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)

	// Create and initialize simulator
	simulator := sim.NewSimulator(rdb, tmRepo, logger)
	if err := simulator.LoadMnemonics(ctx); err != nil {
		logger.Error("failed to load mnemonics", "error", err)
		os.Exit(1)
	}
	if err := simulator.EnsureConfig(ctx); err != nil {
		logger.Error("failed to ensure simulator config", "error", err)
		os.Exit(1)
	}
	if err := simulator.Reset(ctx); err != nil {
		logger.Error("failed to initialize simulator values", "error", err)
		os.Exit(1)
	}

	// Start simulator loop in background
	go simulator.Run(ctx)

	// Start HTTP server
	handler := sim.NewSimulatorHandler(simulator, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		logger.Info("simulator HTTP server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down simulator...")
	server.Close()
	logger.Info("simulator stopped")
}
