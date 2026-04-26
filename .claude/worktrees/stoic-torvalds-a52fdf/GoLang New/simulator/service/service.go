package service

import (
	"context"
	"fmt"
	"net/http"

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

// Run starts the simulator service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg SimulatorConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("simulator: config error: %w", err)
	}
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = 21001
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	sim.SetLogger(logger)

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("simulator: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("simulator: failed to open SQLite: %w", err)
	}
	defer sdb.Close()

	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)
	simulator := sim.NewSimulator(rdb, tmRepo, logger)
	if err := simulator.LoadMnemonics(ctx); err != nil {
		return fmt.Errorf("simulator: failed to load mnemonics: %w", err)
	}
	if err := simulator.EnsureConfig(ctx); err != nil {
		return fmt.Errorf("simulator: failed to ensure simulator config: %w", err)
	}
	if err := simulator.Reset(ctx); err != nil {
		return fmt.Errorf("simulator: failed to initialize simulator values: %w", err)
	}

	go simulator.Run(ctx)

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

	<-ctx.Done()
	logger.Info("shutting down simulator...")
	server.Close()
	logger.Info("simulator stopped")
	return nil
}
