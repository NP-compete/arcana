package db

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate runs all pending up-migrations inside individual transactions.
// It creates a schema_migrations tracking table on first run and skips
// versions that have already been applied.
func Migrate(conn *sql.DB) error {
	if err := ensureMigrationsTable(conn); err != nil {
		return err
	}

	applied, err := appliedVersions(conn)
	if err != nil {
		return err
	}

	upFiles, err := pendingUpFiles(applied)
	if err != nil {
		return err
	}

	for _, f := range upFiles {
		version := strings.TrimSuffix(f, ".up.sql")
		if err := applyMigration(conn, version, f); err != nil {
			return err
		}
		log.Info("migration applied", "version", version)
	}

	return nil
}

// Rollback undoes the most recently applied migration. It returns an error
// if no migrations have been applied or if the corresponding .down.sql file
// is missing.
func Rollback(conn *sql.DB) error {
	if err := ensureMigrationsTable(conn); err != nil {
		return err
	}

	var version string
	err := conn.QueryRow(
		"SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1",
	).Scan(&version)
	if err == sql.ErrNoRows {
		log.Info("no migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("query latest migration: %w", err)
	}

	downFile := version + ".down.sql"
	content, err := migrationFS.ReadFile(filepath.Join("migrations", downFile))
	if err != nil {
		return fmt.Errorf("read down migration %s: %w", downFile, err)
	}

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx for rollback %s: %w", version, err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("execute rollback %s: %w", version, err)
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("remove migration record %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rollback %s: %w", version, err)
	}

	log.Info("migration rolled back", "version", version)
	return nil
}

func ensureMigrationsTable(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func appliedVersions(conn *sql.DB) (map[string]bool, error) {
	rows, err := conn.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration rows: %w", err)
	}
	return applied, nil
}

func pendingUpFiles(applied map[string]bool) ([]string, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".up.sql") {
			version := strings.TrimSuffix(e.Name(), ".up.sql")
			if !applied[version] {
				upFiles = append(upFiles, e.Name())
			}
		}
	}
	sort.Strings(upFiles)
	return upFiles, nil
}

func applyMigration(conn *sql.DB, version, filename string) error {
	content, err := migrationFS.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", version, err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("execute migration %s: %w", version, err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES ($1)", version,
	); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("record migration %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}

	return nil
}
