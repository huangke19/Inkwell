package db

import (
	"database/sql"
	_ "embed"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_init.sql
var initSQL string

//go:embed migrations/002_add_ratings.sql
var addRatingsSQL string

//go:embed migrations/003_add_source.sql
var addSourceSQL string

func applyMigration(db *sql.DB, migrationSQL string) error {
	for _, stmt := range strings.Split(migrationSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

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

	// 逐条执行迁移，忽略"列已存在"错误（SQLite 不支持 IF NOT EXISTS）
	for _, migrationSQL := range []string{addRatingsSQL, addSourceSQL} {
		if err := applyMigration(db, migrationSQL); err != nil {
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
