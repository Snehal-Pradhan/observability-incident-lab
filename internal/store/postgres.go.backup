package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snehal/observability-incident-lab/internal/model"
)

var ErrUnknownCustomer = errors.New("unknown customer")

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *PostgresStore) Insert(ctx context.Context, req model.CreateOrderRequest) (model.Order, error) {
	var order model.Order

	err := s.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, status, total_cents)
		VALUES ($1, $2, $3)
		RETURNING id, customer_id, status, total_cents, created_at
	`, req.CustomerID, req.Status, req.TotalCents).Scan(
		&order.ID,
		&order.CustomerID,
		&order.Status,
		&order.TotalCents,
		&order.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return model.Order{}, ErrUnknownCustomer
		}
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}

	return order, nil
}

func (s *PostgresStore) GetByID(ctx context.Context, id int64) (model.Order, bool, error) {
	var order model.Order

	err := s.pool.QueryRow(ctx, `
		SELECT id, customer_id, status, total_cents, created_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.Status,
		&order.TotalCents,
		&order.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, fmt.Errorf("get order by ID: %w", err)
	}

	return order, true, nil
}

func (s *PostgresStore) ListByCustomer(ctx context.Context, customerID int64) ([]model.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, customer_id, status, total_cents, created_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list orders by customer: %w", err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0)
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.Status,
			&order.TotalCents,
			&order.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}
