package db

import (
	"database/sql"
	_ "embed"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_init.sql
var initSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(initSQL); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
