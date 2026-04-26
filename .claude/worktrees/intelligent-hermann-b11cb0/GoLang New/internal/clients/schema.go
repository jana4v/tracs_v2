package clients

// allSchemas is executed in order during SQLiteDB.migrate().
// Each statement is idempotent (CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).
var allSchemas = []string{
	// ── Telemetry mnemonics ────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS tm_mnemonics (
		id        TEXT NOT NULL PRIMARY KEY,
		subsystem TEXT NOT NULL DEFAULT '',
		data      TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tm_mnemonics_subsystem ON tm_mnemonics(subsystem)`,

	// ── TC mnemonics ──────────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS tc_mnemonics (
		id        TEXT NOT NULL PRIMARY KEY,
		subsystem TEXT NOT NULL DEFAULT '',
		data      TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tc_mnemonics_subsystem ON tc_mnemonics(subsystem)`,

	// ── SCO commands ──────────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS sco_commands (
		id        TEXT NOT NULL PRIMARY KEY,
		subsystem TEXT NOT NULL DEFAULT '',
		data      TEXT NOT NULL
	)`,

	// ── DTM procedures (one document per project) ─────────────────────────────
	`CREATE TABLE IF NOT EXISTS dtm_procedures (
		project TEXT NOT NULL PRIMARY KEY,
		data    TEXT NOT NULL
	)`,

	// ── User-defined telemetry (one document per project) ─────────────────────
	`CREATE TABLE IF NOT EXISTS user_telemetry (
		project TEXT NOT NULL PRIMARY KEY,
		data    TEXT NOT NULL
	)`,

	// ── User-defined telemetry — version snapshots ────────────────────────────
	`CREATE TABLE IF NOT EXISTS user_telemetry_versions (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		project TEXT    NOT NULL,
		version INTEGER NOT NULL,
		data    TEXT    NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_utv_project_version ON user_telemetry_versions(project, version)`,

	// ── Spasdacs diagrams ─────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS spasdacs (
		id   TEXT NOT NULL PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		data TEXT NOT NULL
	)`,

	// ── IAM: users ────────────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS iam_users (
		id       TEXT NOT NULL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		email    TEXT NOT NULL DEFAULT '',
		data     TEXT NOT NULL
	)`,

	// ── IAM: roles ────────────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS iam_roles (
		id   TEXT NOT NULL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		data TEXT NOT NULL
	)`,

	// ── IAM: refresh tokens (scalar columns — no JSON blob) ───────────────────
	`CREATE TABLE IF NOT EXISTS iam_refresh_tokens (
		id         TEXT    NOT NULL PRIMARY KEY,
		user_id    TEXT    NOT NULL,
		token      TEXT    NOT NULL UNIQUE,
		expires_at INTEGER NOT NULL,
		revoked    INTEGER NOT NULL DEFAULT 0
	)`,
	`CREATE INDEX IF NOT EXISTS idx_iam_rt_token    ON iam_refresh_tokens(token)`,
	`CREATE INDEX IF NOT EXISTS idx_iam_rt_user_id  ON iam_refresh_tokens(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_iam_rt_expires  ON iam_refresh_tokens(expires_at)`,

	// ── TM change history (append-only JSON array per mnemonic ID) ────────────
	`CREATE TABLE IF NOT EXISTS tm_mnemonics_change_history (
		id      TEXT NOT NULL PRIMARY KEY,
		history TEXT NOT NULL DEFAULT '[]'
	)`,

	// ── TC change history ─────────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS tc_mnemonics_change_history (
		id      TEXT NOT NULL PRIMARY KEY,
		history TEXT NOT NULL DEFAULT '[]'
	)`,
}
