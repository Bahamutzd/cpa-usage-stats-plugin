// Package sqlite owns the database/sql connection lifecycle for the plugin.
// It uses modernc.org/sqlite (pure Go) so the plugin builds as a c-shared
// library without an extra cgo dependency on the system sqlite.
package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open creates the parent directory, opens the database, and runs migrations.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
