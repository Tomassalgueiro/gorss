package database

import (
	"fmt"
	"database/sql"
	_ "modernc.org/sqlite"
)


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
