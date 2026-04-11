package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/trace-point/trace-point/internal/utils/logger"
)

const (
	// Schema version for migrations
	schemaVersion = 1

	// Default purge days
	defaultPurgeDays = 7
)

// Database represents the SQLite database connection manager.
type Database struct {
	db        *sql.DB
	dbPath    string
	purgeDays int
}

// NewDatabase creates a new database connection with WAL mode.
func NewDatabase(dbPath string, purgeDays int) (*Database, error) {
	// Ensure the directory exists
	dbDir := dbPath[:strings.LastIndex(dbPath, "/")]
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open SQLite connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Enable WAL mode
	if err := enableWALMode(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:        db,
		dbPath:    dbPath,
		purgeDays: purgeDays,
	}

	// Run migrations
	if err := database.runMigrations(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// enableWALMode enables SQLite WAL mode for better concurrency.
func enableWALMode(db *sql.DB) error {
	// Enable WAL mode
	_, err := db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Set synchronous mode to NORMAL for better performance with reasonable durability
	_, err = db.Exec("PRAGMA synchronous=NORMAL")
	if err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Set cache size (negative value represents KB)
	_, err = db.Exec("PRAGMA cache_size=-64000") // 64MB
	if err != nil {
		return fmt.Errorf("failed to set cache size: %w", err)
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set busy timeout
	_, err = db.Exec("PRAGMA busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return nil
}

// runMigrations executes database schema migrations.
func (d *Database) runMigrations(ctx context.Context) error {
	// Create migrations tracking table
	_, err := d.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check current version
	var currentVersion int
	err = d.db.QueryRowContext(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&currentVersion)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if currentVersion >= schemaVersion {
		logger.Debug("Database schema up to date at version %d", currentVersion)
		return nil
	}

	logger.Info("Running migrations from version %d to %d", currentVersion, schemaVersion)

	// Run migrations - execute each migration as a single statement
	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
				CREATE TABLE IF NOT EXISTS spike_events (
					id TEXT PRIMARY KEY,
					timestamp TEXT NOT NULL,
					pod_name TEXT NOT NULL,
					namespace TEXT NOT NULL,
					cpu_usage_percent REAL NOT NULL,
					cpu_limit_percent REAL NOT NULL,
					ram_usage_percent REAL NOT NULL,
					ram_limit_percent REAL NOT NULL,
					threshold_percent REAL NOT NULL,
					moving_average_percent REAL NOT NULL,
					route_name TEXT,
					trace_id TEXT,
					culprit_function TEXT,
					culprit_file_path TEXT,
					alert_sent INTEGER NOT NULL DEFAULT 0,
					cooldown_end TEXT,
					created_at TEXT NOT NULL DEFAULT (datetime('now'))
				);

				CREATE INDEX IF NOT EXISTS idx_spike_timestamp ON spike_events(timestamp);

				CREATE INDEX IF NOT EXISTS idx_spike_pod ON spike_events(pod_name, namespace);

				CREATE INDEX IF NOT EXISTS idx_spike_cooldown ON spike_events(cooldown_end);

				CREATE TABLE IF NOT EXISTS config (
					id TEXT PRIMARY KEY,
					key TEXT NOT NULL UNIQUE,
					value TEXT NOT NULL,
					updated_at TEXT NOT NULL DEFAULT (datetime('now'))
				);

				CREATE INDEX IF NOT EXISTS idx_config_key ON config(key);
			`,
		},
	}

	for _, migration := range migrations {
		if migration.version > currentVersion {
			logger.Debug("Applying migration v%d", migration.version)

			// Execute each statement individually
			statements := strings.Split(migration.sql, ";")
			for _, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				// Skip empty statements and comments
				if stmt == "" || strings.HasPrefix(stmt, "--") {
					continue
				}

				_, err := d.db.ExecContext(ctx, stmt)
				if err != nil {
					return fmt.Errorf("failed to apply migration v%d: %w", migration.version, err)
				}
			}

			// Record migration
			_, err := d.db.ExecContext(ctx,
				"INSERT INTO schema_migrations (version) VALUES (?)",
				migration.version)
			if err != nil {
				return fmt.Errorf("failed to record migration v%d: %w", migration.version, err)
			}

			logger.Info("Applied migration v%d", migration.version)
		}
	}

	return nil
}

// PurgeOldSpikes removes spike events older than the specified number of days.
func (d *Database) PurgeOldSpikes(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		days = defaultPurgeDays
	}

	result, err := d.db.ExecContext(ctx, `
		DELETE FROM spike_events
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`, days)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old spikes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		logger.Info("Purged %d old spike events older than %d days", rowsAffected, days)
	}

	return rowsAffected, nil
}

// RunAutoPurge executes automatic purge on startup.
func (d *Database) RunAutoPurge(ctx context.Context) error {
	days := d.purgeDays
	if days <= 0 {
		days = defaultPurgeDays
	}

	// Run vacuum to reclaim space after purge
	_, err := d.PurgeOldSpikes(ctx, days)
	if err != nil {
		return err
	}

	// Run VACUUM periodically to reclaim space
	_, err = d.db.ExecContext(ctx, "VACUUM")
	if err != nil {
		logger.Warn("Failed to vacuum database: %v", err)
	}

	return nil
}

// DB returns the underlying database connection.
func (d *Database) DB() *sql.DB {
	return d.db
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d.db != nil {
		// Checkpoint WAL before closing
		if _, err := d.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			logger.Warn("Failed to checkpoint WAL: %v", err)
		}
		return d.db.Close()
	}
	return nil
}
