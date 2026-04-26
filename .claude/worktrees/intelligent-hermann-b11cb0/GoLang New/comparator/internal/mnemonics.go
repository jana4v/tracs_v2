package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

// MnemonicLoader loads mnemonics from the store where enable_comparison=true.
// It provides thread-safe access for the comparator loop and supports
// event-driven reloading via TM_LIMIT_CHANGED subscription.
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

// Load performs the initial load of mnemonics from the store.
func (l *MnemonicLoader) Load(ctx context.Context) error {
	return l.loadFromStore(ctx)
}

// Reload reloads mnemonics from the store. Called when TM_LIMIT_CHANGED is received.
func (l *MnemonicLoader) Reload(ctx context.Context) error {
	return l.loadFromStore(ctx)
}

// Get returns the current list of mnemonics (thread-safe).
func (l *MnemonicLoader) Get() []models.TmMnemonic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mnemonics
}

// loadFromStore queries for mnemonics with enable_comparison=true.
func (l *MnemonicLoader) loadFromStore(ctx context.Context) error {
	results, err := l.store.FindWithComparisonEnabled(ctx)
	if err != nil {
		return fmt.Errorf("find mnemonics with enable_comparison=true: %w", err)
	}
	if results == nil {
		results = []models.TmMnemonic{}
	}

	l.mu.Lock()
	l.mnemonics = results
	l.mu.Unlock()

	l.logger.Info("loaded mnemonics", "count", len(results), "filter", "enable_comparison=true")
	return nil
}
