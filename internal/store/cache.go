package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, span := otel.Tracer("orders-api/store").Start(ctx, "redis.get")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("db.operation.name", "GET"))
	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis GET failed")
		return nil, false, fmt.Errorf("get cache key %q: %w", key, err)
	}

	return value, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := otel.Tracer("orders-api/store").Start(ctx, "redis.set")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("db.operation.name", "SET"))
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis SET failed")
		return fmt.Errorf("set cache key %q: %w", key, err)
	}

	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	ctx, span := otel.Tracer("orders-api/store").Start(ctx, "redis.delete")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("db.operation.name", "DEL"))
	if err := c.client.Del(ctx, key).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis DEL failed")
		return fmt.Errorf("delete cache key %q: %w", key, err)
	}

	return nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	ctx, span := otel.Tracer("orders-api/store").Start(ctx, "redis.ping")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("db.operation.name", "PING"))
	if err := c.client.Ping(ctx).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Redis ping failed")
		return err
	}
	return nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
