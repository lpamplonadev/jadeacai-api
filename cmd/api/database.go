package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type orderStore interface {
	Create(context.Context, createOrderRequest) (string, error)
}

type postgresOrderStore struct {
	db *sql.DB
}

func openOrderStore(connectionString string) (*postgresOrderStore, error) {
	if strings.TrimSpace(connectionString) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	var ordersTableExists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.orders') IS NOT NULL").Scan(&ordersTableExists); err != nil {
		db.Close()
		return nil, fmt.Errorf("check orders table: %w", err)
	}
	if !ordersTableExists {
		db.Close()
		return nil, errors.New("orders table does not exist; run the Supabase database migrations")
	}

	return &postgresOrderStore{db: db}, nil
}

func (store *postgresOrderStore) Create(ctx context.Context, request createOrderRequest) (string, error) {
	orderData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode order: %w", err)
	}

	const query = `
		INSERT INTO orders (
			customer_name,
			customer_phone,
			estimated_total_cents,
			order_data
		)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id::text
	`

	var orderID string
	err = store.db.QueryRowContext(
		ctx,
		query,
		request.Customer.Name,
		request.Customer.Phone,
		request.EstimatedTotalCents,
		orderData,
	).Scan(&orderID)
	if err != nil {
		return "", fmt.Errorf("insert order: %w", err)
	}

	return orderID, nil
}

func (store *postgresOrderStore) Close() {
	store.db.Close()
}
