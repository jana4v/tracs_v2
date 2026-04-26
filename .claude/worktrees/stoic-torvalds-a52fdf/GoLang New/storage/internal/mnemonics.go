package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

// MnemonicLoader loads mnemonics from the TMMnemonicStore where enable_storage=true.
// It provides thread-safe access for the storage loop and supports
// event-driven reloading.
type MnemonicLoader struct {
	store  repository.TMMnemonicStore
	logger *slog.Logger

	mu        sync.RWMutex
	mnemonics []models.TmMnemonic
}

// NewMnemonicLoader creates a new MnemonicLoader.
func NewMnemonicLoader(store repository.TMMnemonicStore, logger *slog.Logger) *MnemonicLoader {
	return &MnemonicLoader{
		store:  store,
		logger: logger.With("component", "mnemonic-loader"),
	}
}

// Load performs the initial load of mnemonics.
func (l *MnemonicLoader) Load(ctx context.Context) error {
	return l.loadFromStore(ctx)
}

// Reload reloads mnemonics.
func (l *MnemonicLoader) Reload(ctx context.Context) error {
	return l.loadFromStore(ctx)
}

// Get returns the current list of mnemonics (thread-safe).
func (l *MnemonicLoader) Get() []models.TmMnemonic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mnemonics
}

// loadFromStore fetches all mnemonics and filters to those with enable_storage=true.
func (l *MnemonicLoader) loadFromStore(ctx context.Context) error {
	all, err := l.store.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("find mnemonics with enable_storage=true: %w", err)
	}

	var results []models.TmMnemonic
	for _, m := range all {
		if m.EnableStorage {
			results = append(results, m)
		}
	}

	l.mu.Lock()
	l.mnemonics = results
	l.mu.Unlock()

	l.logger.Info("loaded mnemonics", "count", len(results), "filter", "enable_storage=true")
	return nil
}
