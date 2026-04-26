package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"

	monitor "github.com/mainframe/tm-system/chainmon/internal"
)

// ChainMonConfig extends BaseConfig with chain monitoring settings.
type ChainMonConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	Chains            []ChainEntry `mapstructure:"chains"`
}

// ChainEntry defines a single chain to monitor.
type ChainEntry struct {
	Name           string `mapstructure:"name"`
	Type           string `mapstructure:"type"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// Run starts the chain monitor service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg ChainMonConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("chainmon: config error: %w", err)
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("chainmon: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	publisher := monitor.NewHeartbeatPublisher(rdb, logger)

	var wg sync.WaitGroup
	for _, chain := range cfg.Chains {
		wg.Add(1)
		go func(c ChainEntry) {
			defer wg.Done()
			mon := monitor.NewChainMonitor(c.Name, c.Type, c.TimeoutSeconds, rdb, publisher, logger)
			mon.Run(ctx)
		}(chain)
	}

	logger.Info("chain monitor started", "chains", len(cfg.Chains))

	<-ctx.Done()
	logger.Info("shutting down chain monitor...")
	wg.Wait()
	logger.Info("chain monitor stopped")
	return nil
}
