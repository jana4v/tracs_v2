package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository/sqlite"

	limiter "github.com/mainframe/tm-system/limiter/internal"
)

// LimiterConfig extends BaseConfig with limit monitor settings.
type LimiterConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	PollIntervalMs    int `mapstructure:"poll_interval_ms"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	var cfg LimiterConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if cfg.PollIntervalMs == 0 {
		cfg.PollIntervalMs = 500
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

	// Load mnemonics with enable_limit=true
	loader := limiter.NewMnemonicLoader(tmRepo, logger)
	if err := loader.Load(ctx); err != nil {
		logger.Error("failed to load mnemonics", "error", err)
		os.Exit(1)
	}

	// Subscribe to MDB_TM_MNEMONICS_UPDATED for event-driven reload
	go func() {
		sub := rdb.Subscribe(ctx, models.MdbTmMnemonicsUpdated)
		defer sub.Close()

		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				if msg == nil {
					continue
				}
				logger.Info("received MDB_TM_MNEMONICS_UPDATED, reloading mnemonics")
				if err := loader.Reload(ctx); err != nil {
					logger.Error("failed to reload mnemonics", "error", err)
				}
			}
		}
	}()

	// Create and run monitor
	mon := limiter.NewMonitor(rdb, loader, logger)

	logger.Info("limit monitor started", "poll_interval_ms", cfg.PollIntervalMs)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("limit monitor stopped")
			return
		case <-ticker.C:
			mon.Check(ctx)
		}
	}
}
