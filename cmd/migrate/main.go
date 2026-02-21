package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/config"
)

const migrationsDir = "internal/infrastructure/postgres/migrations"

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.PGDSN)
	if err != nil {
		fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := ensureMigrationsTable(ctx, pool); err != nil {
		fatalf("ensure migrations table: %v", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := runUp(ctx, pool); err != nil {
			fatalf("migrate up: %v", err)
		}
	case "down":
		if err := runDown(ctx, pool); err != nil {
			fatalf("migrate down: %v", err)
		}
	default:
		fatalf("unknown command %q — use 'up' or 'down'", cmd)
	}
}

func runUp(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := listUpMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("no migrations found")
		return nil
	}

	applied, err := loadApplied(ctx, pool)
	if err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}

	appliedCount := 0
	for _, file := range files {
		version := versionFromUpFile(file)
		if applied[version] {
			continue
		}
		if err := applyMigration(ctx, pool, version, file); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}
		appliedCount++
		fmt.Printf("applied %s\n", version)
	}

	if appliedCount == 0 {
		fmt.Println("no new migrations to apply")
	}
	return nil
}

func runDown(ctx context.Context, pool *pgxpool.Pool) error {
	version, err := lastApplied(ctx, pool)
	if err != nil {
		return fmt.Errorf("query last applied: %w", err)
	}
	if version == "" {
		fmt.Println("nothing to roll back")
		return nil
	}

	downFile := filepath.Join(migrationsDir, version+".down.sql")
	if err := rollbackMigration(ctx, pool, version, downFile); err != nil {
		return fmt.Errorf("rollback %s: %w", version, err)
	}
	fmt.Printf("rolled back %s\n", version)
	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func listUpMigrations(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".up.sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func versionFromUpFile(file string) string {
	return strings.TrimSuffix(filepath.Base(file), ".up.sql")
}

func loadApplied(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func lastApplied(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var version string
	err := pool.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return version, err
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)`, version, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func rollbackMigration(ctx context.Context, pool *pgxpool.Pool, version, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read down migration %s: %w", file, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
