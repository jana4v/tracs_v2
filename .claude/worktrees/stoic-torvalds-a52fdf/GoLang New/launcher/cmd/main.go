package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	chainmonsvc   "github.com/mainframe/tm-system/chainmon/service"
	comparatorsvc "github.com/mainframe/tm-system/comparator/service"
	gatewaysvc    "github.com/mainframe/tm-system/gateway/service"
	iamsvc        "github.com/mainframe/tm-system/iam/service"
	ingestsvc     "github.com/mainframe/tm-system/ingest/service"
	limitersvc    "github.com/mainframe/tm-system/limiter/service"
	simulatorsvc  "github.com/mainframe/tm-system/simulator/service"
	storagesvc    "github.com/mainframe/tm-system/storage/service"
	umacstcsvc    "github.com/mainframe/umacs-tc/service"
)

// ServiceConfig describes a single service entry in the launcher config.
type ServiceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Config  string `yaml:"config"`
}

// LauncherConfig is the top-level structure of launcher/config.yaml.
type LauncherConfig struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

// serviceRunner pairs a service name with its Run function.
type serviceRunner struct {
	name string
	run  func(ctx context.Context, configPath string) error
}

// startService launches run in a goroutine and adds to wg.
// Each goroutine is protected by a panic recovery so a misbehaving service
// cannot crash the whole process. Services fail independently; a failing
// service does NOT cancel the shared context for the others.
func startService(ctx context.Context, wg *sync.WaitGroup, name, cfgPath string,
	run func(context.Context, string) error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[launcher] PANIC in %s: %v\n", name, r)
			}
		}()
		fmt.Printf("[launcher] starting %s (config: %s)\n", name, cfgPath)
		if err := run(ctx, cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "[launcher] %s exited with error: %v\n", name, err)
		}
		fmt.Printf("[launcher] %s stopped\n", name)
	}()
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to launcher config file")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[launcher] failed to read config %q: %v\n", *configPath, err)
		os.Exit(1)
	}

	var cfg LauncherConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[launcher] failed to parse config: %v\n", err)
		os.Exit(1)
	}

	// Map service names to their Run functions (order determines startup sequence).
	runners := []serviceRunner{
		{"iam", iamsvc.Run},
		{"chainmon", chainmonsvc.Run},
		{"comparator", comparatorsvc.Run},
		{"gateway", gatewaysvc.Run},
		{"ingest", ingestsvc.Run},
		{"limiter", limitersvc.Run},
		{"simulator", simulatorsvc.Run},
		{"storage", storagesvc.Run},
		{"umacs_tc", umacstcsvc.Run},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for _, r := range runners {
		svcCfg, ok := cfg.Services[r.name]
		if !ok || !svcCfg.Enabled {
			fmt.Printf("[launcher] skipping %s (disabled or not configured)\n", r.name)
			continue
		}
		startService(ctx, &wg, r.name, svcCfg.Config, r.run)
	}

	fmt.Println("[launcher] all enabled services started — waiting for shutdown signal (Ctrl+C / SIGTERM)")
	<-ctx.Done()
	fmt.Println("[launcher] shutdown signal received, waiting for services to stop...")

	// Wait for all goroutines with a hard 30-second timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("[launcher] clean shutdown complete")
	case <-time.After(30 * time.Second):
		fmt.Fprintln(os.Stderr, "[launcher] shutdown timed out after 30s, forcing exit")
		os.Exit(1)
	}
}
