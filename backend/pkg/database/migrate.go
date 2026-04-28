package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	applied := make(map[int]bool)
	var v int
	for rows.Next() {
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}

	// Find migrations directory relative to working dir
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	type migFile struct {
		version int
		name    string
	}
	var files []migFile
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		var ver int
		fmt.Sscanf(e.Name(), "%d", &ver)
		if ver > 0 {
			files = append(files, migFile{version: ver, name: e.Name()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })

	for _, f := range files {
		if applied[f.version] {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationsDir, f.name))
		if err != nil {
			return fmt.Errorf("read %s: %w", f.name, err)
		}
		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("run migration %s: %w", f.name, err)
		}
		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", f.version)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", f.name, err)
		}
	}

	return nil
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"../migrations",
		"backend/migrations",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
