// Package repository defines the persistence interfaces for the IAM subsystem
// (users, roles, and refresh tokens).
//
// Intentional split from internal/repository
//
// The root internal/repository package (github.com/mainframe/tm-system/internal/repository)
// serves the main telemetry domain and owns stores for TM/TC mnemonics, DTM
// procedures, SPASDACS documents, and SCO commands backed by SQLite.
//
// This package is deliberately separate because:
//   - IAM entities (User, Role, RefreshToken) belong to a different bounded context.
//   - The IAM module uses its own SQLite database file with its own schema, allowing
//     it to be deployed or replaced independently of the telemetry pipeline.
//   - Keeping the interfaces in this package avoids a circular import: the iam module
//     depends on internal/models, but internal must not import anything from iam.
//
// Implementations live in iam/internal/repository/<driver>/ (currently sqlite).
package repository
