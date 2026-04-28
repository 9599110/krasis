package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("PG_HOST", "localhost"),
			getEnv("PG_PORT", "5432"),
			getEnv("PG_USER", "postgres"),
			getEnv("PG_PASSWORD", "postgres"),
			getEnv("PG_DBNAME", "krasis"),
			getEnv("PG_SSLMODE", "disable"),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping postgres: %v\n", err)
		os.Exit(1)
	}

	baseline := len(os.Args) > 1 && os.Args[1] == "--baseline"

	if err := run(ctx, pool, baseline); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("migrations up to date")
}

func run(ctx context.Context, pool *pgxpool.Pool, baseline bool) error {
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

	// Baseline mode: if no migrations recorded but tables exist, mark all as applied
	if baseline && len(applied) == 0 {
		for _, f := range files {
			_, err := pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", f.version)
			if err != nil {
				return fmt.Errorf("baseline record %s: %w", f.name, err)
			}
			fmt.Printf("[baseline] %s\n", f.name)
		}
		return nil
	}

	for _, f := range files {
		if applied[f.version] {
			fmt.Printf("[skip] %s\n", f.name)
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
		fmt.Printf("[applied] %s\n", f.name)
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
