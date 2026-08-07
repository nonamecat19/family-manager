package database

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsTable is deliberately golang-migrate's layout, so the `migrate` CLI in
// `just tools` stays usable against a database this runner has touched.
const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT  NOT NULL PRIMARY KEY,
	dirty   BOOLEAN NOT NULL DEFAULT FALSE
)`

// Migration is one numbered up-migration read off disk.
type Migration struct {
	Version int64
	Name    string
	SQL     string
}

// Migrate applies every not-yet-applied `NNNNNN_name.up.sql` in dir, in version order.
// Each migration runs in its own transaction together with its schema_migrations row, so a
// failure leaves the database on the last complete version rather than half-applied.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, dir string) ([]int64, error) {
	migrations, err := LoadMigrations(fsys, dir)
	if err != nil {
		return nil, err
	}

	if _, err := pool.Exec(ctx, migrationsTable); err != nil {
		return nil, fmt.Errorf("database: create schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return nil, err
	}

	var ran []int64
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		if err := applyOne(ctx, pool, m); err != nil {
			return ran, err
		}
		ran = append(ran, m.Version)
	}
	return ran, nil
}

// LoadMigrations parses the up-migrations in dir without touching a database.
func LoadMigrations(fsys fs.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("database: read migrations dir %q: %w", dir, err)
	}

	var out []Migration
	seen := map[int64]string{}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		version, err := parseVersion(name)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("database: duplicate migration version %d (%s and %s)", version, prev, name)
		}
		seen[version] = name

		body, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("database: read %s: %w", name, err)
		}
		out = append(out, Migration{Version: version, Name: name, SQL: string(body)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func parseVersion(filename string) (int64, error) {
	idx := strings.Index(filename, "_")
	if idx <= 0 {
		return 0, fmt.Errorf("database: migration %q is not NNNNNN_name.up.sql", filename)
	}
	v, err := strconv.ParseInt(filename[:idx], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("database: migration %q has a non-numeric version: %w", filename, err)
	}
	return v, nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]struct{}, error) {
	rows, err := pool.Query(ctx, `SELECT version, dirty FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("database: read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int64]struct{}{}
	for rows.Next() {
		var version int64
		var dirty bool
		if err := rows.Scan(&version, &dirty); err != nil {
			return nil, fmt.Errorf("database: scan schema_migrations: %w", err)
		}
		if dirty {
			return nil, fmt.Errorf("database: migration %d is dirty; resolve it before booting", version)
		}
		applied[version] = struct{}{}
	}
	return applied, rows.Err()
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database: begin %s: %w", m.Name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return fmt.Errorf("database: apply %s: %w", m.Name, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version, dirty) VALUES ($1, FALSE)`, m.Version,
	); err != nil {
		return fmt.Errorf("database: record %s: %w", m.Name, err)
	}
	return commit(ctx, tx, m)
}

func commit(ctx context.Context, tx pgx.Tx, m Migration) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit %s: %w", m.Name, err)
	}
	return nil
}
