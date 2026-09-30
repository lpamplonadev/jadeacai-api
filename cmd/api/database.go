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
	List(context.Context, orderListFilter) (paginatedOrders, error)
	UpdateStatus(context.Context, string, string) (bool, error)
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

func (store *postgresOrderStore) List(ctx context.Context, filter orderListFilter) (paginatedOrders, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM orders
		WHERE ($1 = '' OR status = $1)
			AND ($2 = '' OR customer_name ILIKE '%' || $2 || '%' OR customer_phone LIKE '%' || $2 || '%')
	`

	result := paginatedOrders{
		Orders: make([]storedOrder, 0, filter.Limit),
		Page:   filter.Page,
		Limit:  filter.Limit,
	}
	if err := store.db.QueryRowContext(ctx, countQuery, filter.Status, filter.Search).Scan(&result.Total); err != nil {
		return paginatedOrders{}, fmt.Errorf("count orders: %w", err)
	}

	const listQuery = `
		SELECT id::text, status, customer_name, customer_phone, estimated_total_cents, order_data, created_at
		FROM orders
		WHERE ($1 = '' OR status = $1)
			AND ($2 = '' OR customer_name ILIKE '%' || $2 || '%' OR customer_phone LIKE '%' || $2 || '%')
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := store.db.QueryContext(ctx, listQuery, filter.Status, filter.Search, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return paginatedOrders{}, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var order storedOrder
		var orderData []byte
		if err := rows.Scan(
			&order.ID,
			&order.Status,
			&order.CustomerName,
			&order.CustomerPhone,
			&order.EstimatedTotalCents,
			&orderData,
			&order.CreatedAt,
		); err != nil {
			return paginatedOrders{}, fmt.Errorf("scan order: %w", err)
		}
		order.OrderData = json.RawMessage(orderData)
		result.Orders = append(result.Orders, order)
	}
	if err := rows.Err(); err != nil {
		return paginatedOrders{}, fmt.Errorf("iterate orders: %w", err)
	}

	return result, nil
}

func (store *postgresOrderStore) UpdateStatus(ctx context.Context, orderID, status string) (bool, error) {
	const query = `
		UPDATE orders
		SET status = $1
		WHERE id = $2::uuid
		RETURNING id
	`

	var updatedID string
	err := store.db.QueryRowContext(ctx, query, status, orderID).Scan(&updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("update order status: %w", err)
	}
	return true, nil
}

func (store *postgresOrderStore) Close() {
	store.db.Close()
}
