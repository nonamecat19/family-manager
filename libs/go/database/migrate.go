package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version  BIGINT  NOT NULL PRIMARY KEY,
	dirty    BOOLEAN NOT NULL DEFAULT FALSE,
	checksum TEXT
)`

const migrationsChecksum = `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT`

type Migration struct {
	Version  int64
	Name     string
	SQL      string
	Checksum string
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, dir string) ([]int64, error) {
	migrations, err := LoadMigrations(fsys, dir)
	if err != nil {
		return nil, err
	}

	if _, err := pool.Exec(ctx, migrationsTable); err != nil {
		return nil, fmt.Errorf("database: create schema_migrations: %w", err)
	}
	if _, err := pool.Exec(ctx, migrationsChecksum); err != nil {
		return nil, fmt.Errorf("database: add schema_migrations.checksum: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return nil, err
	}
	unrecorded, ahead, err := CheckApplied(applied, migrations)
	if err != nil {
		return nil, err
	}
	if len(ahead) > 0 {
		slog.Default().WarnContext(ctx, "database has migrations newer than this build; booting on the assumption of a rollback",
			slog.Any("versions", ahead))
	}
	if err := recordChecksums(ctx, pool, unrecorded); err != nil {
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
		out = append(out, Migration{Version: version, Name: name, SQL: string(body), Checksum: Checksum(body)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func Checksum(body []byte) string {
	sum := sha256.Sum256([]byte(strings.ReplaceAll(string(body), "\r\n", "\n")))
	return hex.EncodeToString(sum[:])
}

func CheckApplied(applied map[int64]string, migrations []Migration) ([]Migration, []int64, error) {
	byVersion := make(map[int64]Migration, len(migrations))
	var latest int64
	for _, m := range migrations {
		byVersion[m.Version] = m
		latest = max(latest, m.Version)
	}

	versions := make([]int64, 0, len(applied))
	for v := range applied {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	if len(versions) > 0 && versions[0] > latest {
		return nil, nil, fmt.Errorf(
			"database: every recorded migration (lowest %d) is newer than this build's highest (%d); "+
				"this build does not know the database's history, so it will not run its migrations on top of it",
			versions[0], latest)
	}

	var unrecorded []Migration
	var ahead []int64
	for _, v := range versions {
		if v > latest {
			ahead = append(ahead, v)
			continue
		}
		m, ok := byVersion[v]
		if !ok {
			return nil, nil, fmt.Errorf(
				"database: migration %d is recorded as applied but its .up.sql file is gone; "+
					"an applied migration was deleted or renumbered, so this database does not have "+
					"the schema the code expects. Add a new migration instead of rewriting history. If the row was "+
					"left by a failed release that was rolled back, revert that release's schema change and run "+
					"DELETE FROM schema_migrations WHERE version = %d", v, v)
		}
		recorded := applied[v]
		switch {
		case recorded == "":
			unrecorded = append(unrecorded, m)
		case recorded != m.Checksum:
			return nil, nil, fmt.Errorf(
				"database: %s changed after it was applied (recorded sha256 %s, file sha256 %s); "+
					"the edit never ran here. Add a new migration instead. If the edit is cosmetic, run "+
					"UPDATE schema_migrations SET checksum = '%s' WHERE version = %d",
				m.Name, recorded, m.Checksum, m.Checksum, v)
		}
	}
	return unrecorded, ahead, nil
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

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]string, error) {
	rows, err := pool.Query(ctx, `SELECT version, dirty, COALESCE(checksum, '') FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("database: read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int64]string{}
	for rows.Next() {
		var version int64
		var dirty bool
		var checksum string
		if err := rows.Scan(&version, &dirty, &checksum); err != nil {
			return nil, fmt.Errorf("database: scan schema_migrations: %w", err)
		}
		if dirty {
			return nil, fmt.Errorf("database: migration %d is dirty; resolve it before booting", version)
		}
		applied[version] = checksum
	}
	return applied, rows.Err()
}

func recordChecksums(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	for _, m := range migrations {
		if _, err := pool.Exec(ctx,
			`UPDATE schema_migrations SET checksum = $2 WHERE version = $1 AND checksum IS NULL`,
			m.Version, m.Checksum,
		); err != nil {
			return fmt.Errorf("database: record checksum of %s: %w", m.Name, err)
		}
	}
	return nil
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
		`INSERT INTO schema_migrations (version, dirty, checksum) VALUES ($1, FALSE, $2)`, m.Version, m.Checksum,
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
