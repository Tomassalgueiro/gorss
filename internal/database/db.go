package database

import (
	"database/sql"
	_ "modernc.org/sqlite"
)


func Open(dbPath string) (*sql.DB, error) {

	db, err := sql.Open("sqlite", dbPath)

	_, err = db.Exec("PRAGMA journal_mode = WAL")
	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	_, err = db.Exec("PRAGMA foreign_keys = ON")

	db.Ping()

	if err == nil {
		return db, nil
	}
	return db, err

}
