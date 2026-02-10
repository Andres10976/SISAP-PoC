package database

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var migrationSQL string

func Migrate(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	return nil
}
