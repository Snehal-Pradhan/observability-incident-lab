package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snehal/observability-incident-lab/internal/config"
	"github.com/snehal/observability-incident-lab/internal/store"
	"github.com/snehal/observability-incident-lab/migrations"
)

const (
	customerCount = 10_000
	orderCount    = 1_000_000
)

func main() {
	force := flag.Bool("force", false, "truncate existing data and reload")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := store.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	exists, err := seeded(ctx, pool)
	if err != nil {
		logger.Error("check existing data failed", "error", err)
		os.Exit(1)
	}
	if *force {
		if _, err := pool.Exec(ctx, `TRUNCATE orders, customers RESTART IDENTITY`); err != nil {
			logger.Error("truncate failed", "error", err)
			os.Exit(1)
		}
		logger.Warn("existing data truncated")
	} else if exists {
		logger.Error("database already contains data; refusing to seed (use --force to truncate and reload)")
		os.Exit(1)
	}

	start := time.Now()

	n, err := insertCustomers(ctx, pool)
	if err != nil {
		logger.Error("insert customers failed", "error", err)
		os.Exit(1)
	}

	m, err := insertOrders(ctx, pool)
	if err != nil {
		logger.Error("insert orders failed", "error", err)
		os.Exit(1)
	}

	logger.Info("seed complete",
		"customers", n,
		"orders", m,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func seeded(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM customers LIMIT 1) OR EXISTS (SELECT 1 FROM orders LIMIT 1)
	`).Scan(&exists)
	return exists, err
}

func insertCustomers(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, email, created_at)
		SELECT
			g,
			'Customer ' || g,
			'customer' || g || '@example.com',
			now() - (random() * interval '730 days')
		FROM generate_series(1, $1) AS g
	`, customerCount)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func insertOrders(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `
		INSERT INTO orders (customer_id, status, total_cents, created_at)
		SELECT
			1 + (g % $1),
			CASE g % 5
				WHEN 0 THEN 'pending'
				WHEN 1 THEN 'paid'
				WHEN 2 THEN 'shipped'
				WHEN 3 THEN 'cancelled'
				ELSE 'refunded'
			END,
			1 + (random() * 100000)::bigint,
			now() - (random() * interval '730 days')
		FROM generate_series(1, $2) AS g
	`, customerCount, orderCount)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
