package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

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

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	var cfg ChainMonConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
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

	// Create heartbeat publisher
	publisher := monitor.NewHeartbeatPublisher(rdb, logger)

	// Launch a monitor goroutine per chain
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

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down chain monitor...")
	wg.Wait()
	logger.Info("chain monitor stopped")
}
