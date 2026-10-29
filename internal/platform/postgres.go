package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
    pool, err := pgxpool.New(ctx, url)
    if err != nil {
        return nil, err
    }
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("postgres ping: %w", err)
    }
    return pool, nil
}