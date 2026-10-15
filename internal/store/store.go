package store

import (
	"context"
	"time"

	"github.com/snehal/observability-incident-lab/internal/model"
)

// OrderStore defines persistence operations for orders.
type OrderStore interface {
	Insert(ctx context.Context, req model.CreateOrderRequest) (model.Order, error)
	GetByID(ctx context.Context, id int64) (model.Order, bool, error)
	ListByCustomer(ctx context.Context, customerID int64) ([]model.Order, error)
}

// Pinger checks whether a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Cache defines the operations required by the cache-aside handlers.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

// Compile-time interface conformance checks.
var (
	_ OrderStore = (*PostgresStore)(nil)
	_ Pinger     = (*PostgresStore)(nil)
	_ Cache      = (*RedisCache)(nil)
	_ Pinger     = (*RedisCache)(nil)
)
