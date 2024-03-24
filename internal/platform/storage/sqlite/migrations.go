package sqlite

import (
	"database/sql"
	"fmt"
)

type migration struct {
	id   int
	name string
	sql  string
}

var migrations = []migration{
	{
		id:   1,
		name: "create_track_metadata",
		sql: `
CREATE TABLE IF NOT EXISTS track_metadata (
  video_id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  title TEXT,
  artist TEXT,
  album TEXT,
  cover_url TEXT,
  duration_sec INTEGER,
  fetched_at DATETIME NOT NULL,
  last_accessed_at DATETIME NOT NULL,
  source_version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_track_metadata_last_accessed_at
  ON track_metadata(last_accessed_at);

CREATE INDEX IF NOT EXISTS idx_track_metadata_fetched_at
  ON track_metadata(fetched_at);
`,
	},
}

func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	applied := make(map[int]struct{})
	rows, err := db.Query("SELECT id FROM schema_migrations;")
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate applied migrations: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, m := range migrations {
		if _, ok := applied[m.id]; ok {
			continue
		}
		if _, err := tx.Exec(m.sql); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.id, m.name, err)
		}
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations(id, name) VALUES(?, ?);",
			m.id,
			m.name,
		); err != nil {
			return fmt.Errorf("record migration %d (%s): %w", m.id, m.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}

	return nil
}
