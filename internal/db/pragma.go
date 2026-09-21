package db

import (
	"context"
	"database/sql"
	"fmt"
)

// ApplyImportPragmas sets import-only PRAGMA on one write connection.
func ApplyImportPragmas(ctx context.Context, conn *sql.Conn) error {
	stmts := []string{
		"PRAGMA foreign_keys=OFF",
		fmt.Sprintf("PRAGMA cache_size=%d", importCacheSize),
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
	}
	for _, s := range stmts {
		if _, err := conn.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// RestoreWorkPragmas returns a write connection to the DSN working values.
func RestoreWorkPragmas(ctx context.Context, conn *sql.Conn) error {
	stmts := []string{
		"PRAGMA foreign_keys=ON",
		fmt.Sprintf("PRAGMA cache_size=%d", workingCacheSize),
		"PRAGMA synchronous=FULL",
		"PRAGMA temp_store=MEMORY",
	}
	for _, s := range stmts {
		if _, err := conn.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// ReadPragma is exported for import tests that check pool isolation.
func ReadPragma(ctx context.Context, pool *sql.DB, name string) (string, error) {
	return readPragma(ctx, pool, name)
}
