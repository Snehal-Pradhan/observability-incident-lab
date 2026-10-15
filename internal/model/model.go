package model

import (
	"errors"
	"time"
)

type Customer struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Order struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	Status     string    `json:"status"`
	TotalCents int64     `json:"total_cents"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	CustomerID int64  `json:"customer_id"`
	Status     string `json:"status"`
	TotalCents int64  `json:"total_cents"`
}

func (r CreateOrderRequest) Validate() error {
	if r.CustomerID <= 0 {
		return errors.New("customer_id must be greater than zero")
	}

	if r.TotalCents <= 0 {
		return errors.New("total_cents must be greater than zero")
	}

	switch r.Status {
	case "pending", "paid", "shipped", "cancelled", "refunded":
		return nil
	default:
		return errors.New("invalid order status")
	}
}
