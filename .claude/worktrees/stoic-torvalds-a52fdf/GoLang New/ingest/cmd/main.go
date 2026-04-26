package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"

	ingest "github.com/mainframe/tm-system/ingest/internal"
)

// IngestConfig extends BaseConfig with chain definitions, WebSocket settings,
// NATS config, and the simulation API port.
type IngestConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	WebSocket         config.WebSocketConfig `mapstructure:"websocket"`
	Chains            []config.ChainConfig   `mapstructure:"chains"`
	NATS              config.NATSConfig      `mapstructure:"nats"`
	SimPort           int                    `mapstructure:"sim_port"` // default 8082
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	var cfg IngestConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if cfg.SimPort == 0 {
		cfg.SimPort = 8082
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- Redis (optional — sim API works without it) ---
	// Redis is only required by the StreamDispatcher (UNIFIED_TM_MAP writes +
	// NATS publishing). The sim HTTP API, SimManager, and chain subscribers all
	// operate entirely in-memory, so the ingest service remains functional for
	// value injection even when Redis is temporarily unavailable.
	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Warn("Redis unavailable — dispatcher disabled, sim API still running", "error", err)
		rdb = nil
	} else {
		defer rdb.Close()
	}

	// --- StreamStore: shared in-memory telemetry registry ---
	store := ingest.NewStreamStore()

	// Pre-create streams for every known chain so the sim API can inject values
	// immediately without waiting for a chain subscriber to connect first.
	// TM1–TM4 and SMON/ADC streams are "programmatic" until a real subscriber
	// attaches; the StreamBuffer is shared, so the subscriber's Update() calls
	// simply overwrite the injected values once live data arrives.
	for _, cc := range cfg.Chains {
		store.GetOrCreate(ingest.StreamMeta{ID: cc.Name, ChainType: "programmatic", ChainName: cc.Name})
	}
	// DTM and UDTM are always programmatic (no WebSocket source).
	store.GetOrCreate(ingest.StreamMeta{ID: "DTM",  ChainType: "programmatic", ChainName: "DTM"})
	store.GetOrCreate(ingest.StreamMeta{ID: "UDTM", ChainType: "programmatic", ChainName: "UDTM"})

	// --- Chain names for heartbeat status publishing ---
	chainNames := make([]string, 0, len(cfg.Chains))
	for _, cc := range cfg.Chains {
		chainNames = append(chainNames, cc.Name)
	}

	// --- NATS (optional: warn and continue if broker is unreachable) ---
	var natsClient *clients.NATSClient
	if nc, err := clients.NewNATSClient(cfg.NATS.URL, cfg.NATS.Name, logger); err != nil {
		logger.Warn("NATS unavailable — TM publishing and status updates disabled", "error", err)
	} else {
		natsClient = nc
		defer natsClient.Close()
	}

	// --- StreamDispatcher: the single NATS publisher ---
	// Only start if Redis is available — it requires Redis for UNIFIED_TM_MAP writes.
	if rdb != nil {
		dispatcher := ingest.NewStreamDispatcher(store, rdb, natsClient, cfg.NATS, logger, chainNames)
		go dispatcher.Run(ctx)
	} else {
		logger.Warn("stream dispatcher not started — Redis is required for NATS publishing")
	}

	// --- SimManager: manages backend random-generation goroutines ---
	simManager := ingest.NewSimManager(store, logger)

	// --- Sim API: HTTP server on sim_port ---
	simMux := ingest.NewSimMux(store, simManager, ctx, logger)
	simAddr := fmt.Sprintf(":%d", cfg.SimPort)
	simSrv := &http.Server{Addr: simAddr, Handler: simMux}
	go func() {
		logger.Info("sim API started", "addr", simAddr)
		if err := simSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("sim API server error", "error", err)
		}
	}()

	if len(cfg.Chains) == 0 {
		logger.Warn("no chains configured — running sim API only")
	}

	// --- Chain subscribers (one goroutine per chain) ---
	var wg sync.WaitGroup
	for _, chain := range cfg.Chains {
		wg.Add(1)
		go func(cc config.ChainConfig) {
			defer wg.Done()
			chainLogger := logger.With("chain", cc.Name, "type", cc.Type)
			sub := ingest.NewChainSubscriber(cc, rdb, store, chainLogger)
			if err := sub.Run(ctx); err != nil && ctx.Err() == nil {
				chainLogger.Error("chain subscriber exited with error", "error", err)
			}
		}(chain)
	}

	slog.Info("tm-ingest started", "chains", len(cfg.Chains), "sim_port", cfg.SimPort)

	<-ctx.Done()
	logger.Info("shutting down tm-ingest...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := simSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("sim server shutdown error", "error", err)
	}

	wg.Wait()
	logger.Info("tm-ingest stopped")
}
