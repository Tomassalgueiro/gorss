package database

import (
	"fmt"
	"database/sql"
	_ "modernc.org/sqlite"
	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

func Open(dbPath string) (*sql.DB, error) {

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("can't connect to db: %w", err)
	}
	db.SetMaxOpenConns(25)

	_, err = db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		return nil, fmt.Errorf("enable wal: %w", err)
	}

	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	if err != nil {
		return nil, fmt.Errorf("database timeout: %w", err)

	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, fmt.Errorf("foreign keys %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("can't ping db %w", err)
	}

	return db, nil

}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Databases created before folders existed need the column added by hand.
	var hasFolderID int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('feeds') WHERE name = 'folder_id'").Scan(&hasFolderID)
	if err != nil {
		return fmt.Errorf("inspect feeds table: %w", err)
	}
	if hasFolderID == 0 {
		_, err = db.Exec("ALTER TABLE feeds ADD COLUMN folder_id INTEGER REFERENCES folders(id) ON DELETE SET NULL")
		if err != nil {
			return fmt.Errorf("add feeds.folder_id: %w", err)
		}
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_folder_id ON feeds(folder_id)")
	if err != nil {
		return fmt.Errorf("create folder index: %w", err)
	}

	return nil
}
