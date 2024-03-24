package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Config struct {
	Path        string
	BusyTimeout time.Duration
	JournalMode string
	Synchronous string
	ForeignKeys bool
}

func DefaultConfig(path string) Config {
	return Config{
		Path:        path,
		BusyTimeout: 5 * time.Second,
		JournalMode: "WAL",
		Synchronous: "NORMAL",
		ForeignKeys: true,
	}
}

func Open(cfg Config) (*sql.DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := applyPragmas(db, cfg); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}

func applyPragmas(db *sql.DB, cfg Config) error {
	busyTimeoutMs := cfg.BusyTimeout.Milliseconds()
	if busyTimeoutMs <= 0 {
		busyTimeoutMs = 5000
	}

	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d;", busyTimeoutMs)); err != nil {
		return fmt.Errorf("set busy_timeout pragma: %w", err)
	}

	if cfg.JournalMode != "" {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA journal_mode = %s;", cfg.JournalMode)); err != nil {
			return fmt.Errorf("set journal_mode pragma: %w", err)
		}
	}

	if cfg.Synchronous != "" {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA synchronous = %s;", cfg.Synchronous)); err != nil {
			return fmt.Errorf("set synchronous pragma: %w", err)
		}
	}

	if cfg.ForeignKeys {
		if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			return fmt.Errorf("set foreign_keys pragma: %w", err)
		}
	}

	return nil
}
