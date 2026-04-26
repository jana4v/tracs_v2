// Package repository defines store interfaces for all domain entities.
// Each interface abstracts the persistence layer so handlers and services
// depend on the interface rather than on a specific database driver.
//
// Implementations live in internal/repository/sqlite/.
package repository

import (
	"context"
	"errors"

	"github.com/mainframe/tm-system/internal/models"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// TMMnemonicStore is the read/write interface for TM mnemonics.
type TMMnemonicStore interface {
	// FindAll returns every mnemonic in the catalogue.
	FindAll(ctx context.Context) ([]models.TmMnemonic, error)

	// FindBySubsystem returns mnemonics whose subsystem matches exactly (case-insensitive).
	FindBySubsystem(ctx context.Context, subsystem string) ([]models.TmMnemonic, error)

	// FindBySubsystemPattern returns mnemonics whose subsystem matches a LIKE pattern,
	// e.g. "%PAY%" for partial match. The caller is responsible for adding % wildcards.
	FindBySubsystemPattern(ctx context.Context, pattern string) ([]models.TmMnemonic, error)

	// FindBySubsystemAndMnemonic returns the single mnemonic whose subsystem and
	// cdbMnemonic both match (case-insensitive). Returns ErrNotFound if missing.
	FindBySubsystemAndMnemonic(ctx context.Context, subsystem, mnemonic string) (*models.TmMnemonic, error)

	// GetSubsystems returns the distinct list of subsystem names.
	GetSubsystems(ctx context.Context) ([]string, error)

	// FindWithComparisonEnabled returns mnemonics that have enable_comparison=true.
	FindWithComparisonEnabled(ctx context.Context) ([]models.TmMnemonic, error)

	// GetByIDRaw returns the raw document map for a given primary-key id.
	// Returns ErrNotFound if no row exists.
	GetByIDRaw(ctx context.Context, id string) (map[string]any, error)

	// SaveDoc upserts a document (INSERT … ON CONFLICT DO UPDATE).
	SaveDoc(ctx context.Context, id, subsystem string, doc map[string]any) error

	// PatchBySubsystemMnemonic performs a read-modify-write on the document
	// identified by (subsystem, cdbMnemonic). patchFn receives the decoded map
	// and may modify it in-place. Returns the number of rows matched (0 or 1).
	PatchBySubsystemMnemonic(ctx context.Context, subsystem, mnemonic string, patchFn func(map[string]any)) (int64, error)

	// AppendHistory appends entry to the tm_mnemonics_change_history for id.
	AppendHistory(ctx context.Context, id string, entry any) error
}

// TCMnemonicStore is the read/write interface for TC mnemonics.
type TCMnemonicStore interface {
	// FindAll returns every TC mnemonic document as a raw map.
	FindAll(ctx context.Context) ([]map[string]any, error)

	// FindBySubsystem returns documents whose subsystem matches (case-insensitive).
	FindBySubsystem(ctx context.Context, subsystem string) ([]map[string]any, error)

	// GetSubsystems returns the distinct list of subsystem names.
	GetSubsystems(ctx context.Context) ([]string, error)

	// GetAllCmdDescs returns all non-null cmdDesc values across all subsystems.
	GetAllCmdDescs(ctx context.Context) ([]string, error)

	// GetCmdDescsBySubsystem returns the cmdDesc values for a subsystem (LIKE match).
	GetCmdDescsBySubsystem(ctx context.Context, subsystem string) ([]string, error)

	// FindByCmdDesc returns the document whose cmdDesc matches (case-insensitive).
	// Returns ErrNotFound if missing.
	FindByCmdDesc(ctx context.Context, cmdDesc string) (map[string]any, error)

	// GetByIDRaw returns the raw document map for a given primary-key id.
	// Returns ErrNotFound if no row exists.
	GetByIDRaw(ctx context.Context, id string) (map[string]any, error)

	// SaveDoc upserts a document (INSERT … ON CONFLICT DO UPDATE).
	SaveDoc(ctx context.Context, id, subsystem string, doc map[string]any) error

	// AppendHistory appends entry to tc_mnemonics_change_history for id.
	AppendHistory(ctx context.Context, id string, entry any) error
}

// SCOCommandStore is the read-only interface for SCO commands.
type SCOCommandStore interface {
	FindAll(ctx context.Context) ([]models.ScoCommand, error)
}

// DTMStore is the read/write interface for DTM procedures.
type DTMStore interface {
	// Get returns the DTM procedures document for a project.
	// Returns ErrNotFound if no document exists.
	Get(ctx context.Context, project string) (*models.DTMProcedures, error)

	// GetRaw returns the raw JSON map for a project.
	// Returns ErrNotFound if no document exists.
	GetRaw(ctx context.Context, project string) (map[string]any, error)

	// Save upserts the procedures document (INSERT … ON CONFLICT DO UPDATE).
	Save(ctx context.Context, doc models.DTMProcedures) error
}

// UDTMStore is the read/write interface for user-defined telemetry.
type UDTMStore interface {
	// Get returns the current UD_TM document for a project.
	// Returns ErrNotFound if no document exists.
	Get(ctx context.Context, project string) (*models.UserTelemetry, error)

	// GetRaw returns the raw JSON map for a project.
	// Returns ErrNotFound if no document exists.
	GetRaw(ctx context.Context, project string) (map[string]any, error)

	// Save upserts the current document (INSERT … ON CONFLICT DO UPDATE).
	Save(ctx context.Context, doc models.UserTelemetry) error

	// SaveVersion inserts a historical version snapshot.
	SaveVersion(ctx context.Context, ver models.UserTelemetryVersion) error

	// ListVersions returns all version summaries for a project.
	ListVersions(ctx context.Context, project string) ([]models.UserTelemetryVersion, error)

	// GetVersion returns a specific version by project + version number.
	// Returns ErrNotFound if missing.
	GetVersion(ctx context.Context, project string, version int) (*models.UserTelemetryVersion, error)
}

// SpasdacsStore is the read/write interface for SPASDACS mimic diagrams.
type SpasdacsStore interface {
	// List returns lightweight metadata for all diagrams, ordered by updatedAt DESC.
	List(ctx context.Context) ([]models.SpasdacsMeta, error)

	// Get returns the full diagram including ModelData.
	// Returns ErrNotFound if missing.
	Get(ctx context.Context, id string) (*models.SpasdacsDiagram, error)

	// Save upserts a diagram (INSERT … ON CONFLICT DO UPDATE).
	Save(ctx context.Context, d models.SpasdacsDiagram) error

	// Patch performs a read-modify-write: patchFn receives the decoded map and
	// may modify it in-place. Returns ErrNotFound if the diagram does not exist.
	Patch(ctx context.Context, id string, patchFn func(map[string]any)) error

	// Delete removes a diagram by id. Returns (true, nil) if deleted,
	// (false, nil) if not found, or (false, err) on error.
	Delete(ctx context.Context, id string) (bool, error)
}
