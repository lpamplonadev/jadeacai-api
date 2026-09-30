CREATE TABLE IF NOT EXISTS order_daily_counters (
    order_date date PRIMARY KEY,
    last_number integer NOT NULL CHECK (last_number > 0)
);

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS order_date date,
    ADD COLUMN IF NOT EXISTS order_number integer;

WITH numbered_orders AS (
    SELECT
        id,
        (created_at AT TIME ZONE 'America/Sao_Paulo')::date AS order_date,
        ROW_NUMBER() OVER (
            PARTITION BY (created_at AT TIME ZONE 'America/Sao_Paulo')::date
            ORDER BY created_at, id
        )::integer AS order_number
    FROM orders
)
UPDATE orders AS existing
SET
    order_date = numbered.order_date,
    order_number = numbered.order_number
FROM numbered_orders AS numbered
WHERE existing.id = numbered.id
    AND (existing.order_date IS NULL OR existing.order_number IS NULL);

ALTER TABLE orders
    ALTER COLUMN order_date SET NOT NULL,
    ALTER COLUMN order_number SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS orders_order_date_number_idx
    ON orders (order_date, order_number);

INSERT INTO order_daily_counters (order_date, last_number)
SELECT order_date, MAX(order_number)
FROM orders
GROUP BY order_date
ON CONFLICT (order_date) DO UPDATE
SET last_number = GREATEST(order_daily_counters.last_number, EXCLUDED.last_number);

ALTER TABLE order_daily_counters ENABLE ROW LEVEL SECURITY;

CREATE OR REPLACE FUNCTION assign_daily_order_number()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    assigned_date date;
    assigned_number integer;
BEGIN
    IF NEW.order_date IS NULL OR NEW.order_number IS NULL THEN
        assigned_date := (COALESCE(NEW.created_at, CURRENT_TIMESTAMP) AT TIME ZONE 'America/Sao_Paulo')::date;

        INSERT INTO order_daily_counters (order_date, last_number)
        VALUES (assigned_date, 1)
        ON CONFLICT (order_date) DO UPDATE
        SET last_number = order_daily_counters.last_number + 1
        RETURNING last_number INTO assigned_number;

        NEW.order_date := assigned_date;
        NEW.order_number := assigned_number;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS orders_assign_daily_number ON orders;
CREATE TRIGGER orders_assign_daily_number
    BEFORE INSERT ON orders
    FOR EACH ROW
    EXECUTE FUNCTION assign_daily_order_number();