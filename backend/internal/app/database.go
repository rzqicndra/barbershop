package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Migration holds one advisory lock so two deployment jobs apply each file once.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock(718204911)"); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock(718204911)")
	if _, err = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		return err
	}
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		var exists bool
		if err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sql, readErr := migrations.ReadFile(name)
		if readErr != nil {
			return readErr
		}
		tx, beginErr := conn.Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		_, err = tx.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations(name) VALUES($1)", name)
		}
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
