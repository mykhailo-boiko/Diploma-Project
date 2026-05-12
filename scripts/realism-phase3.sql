
BEGIN;

WITH customers AS (
    SELECT DISTINCT customer_name FROM orders.orders LIMIT 100
),
product_pool AS (
    SELECT id::text AS pid, name, unit_price, row_number() OVER () AS rn,
           count(*) OVER () AS total
    FROM inventory.product
),
days AS (
    SELECT gs.day, idx
    FROM generate_series(
        date_trunc('day', NOW() - interval '11 months'),
        date_trunc('day', NOW() - interval '1 month'),
        interval '1 day'
    ) gs(day)
    CROSS JOIN LATERAL generate_series(1, 5 + (abs(hashtextextended(gs.day::text, 51) % 6))::int) idx
),
inserted_orders AS (
    INSERT INTO orders.orders (id, customer_name, status, total_amount, created_at, updated_at)
    SELECT gen_random_uuid(),
           (SELECT customer_name FROM customers
            ORDER BY md5(d.day::text || d.idx::text || '7') LIMIT 1),
           CASE
               WHEN (abs(hashtextextended(d.day::text || d.idx::text, 52) % 100)) < 5 THEN 'cancelled'
               WHEN d.day < NOW() - interval '30 days' THEN 'completed'
               WHEN (abs(hashtextextended(d.day::text || d.idx::text, 53) % 100)) < 80 THEN 'delivered'
               ELSE 'shipped'
           END::varchar,
           0,
           d.day - (random() * interval '14 hours'),
           d.day - (random() * interval '14 hours')
    FROM days d
    RETURNING id, created_at
),
new_items AS (
    INSERT INTO orders.order_items (id, order_id, product_id, name, quantity, unit_price, subtotal)
    SELECT gen_random_uuid(),
           io.id,
           pp.pid, pp.name,
           (1 + (abs(hashtextextended(io.id::text || line.idx::text, 61) % 12)))::int,
           ROUND((pp.unit_price * (0.85 + (abs(hashtextextended(io.id::text || line.idx::text, 62) % 1000)::numeric / 1000.0) * 0.30))::numeric, 2),
           0::numeric
    FROM inserted_orders io
    CROSS JOIN LATERAL generate_series(1, 1 + (abs(hashtextextended(io.id::text, 63) % 4))::int) line(idx)
    CROSS JOIN LATERAL (
        SELECT * FROM product_pool
        WHERE rn = ((abs(hashtextextended(io.id::text || line.idx::text, 64)) % total) + 1)
    ) pp
    RETURNING order_id
)
SELECT count(*) AS new_items_inserted FROM new_items;

UPDATE orders.order_items SET subtotal = ROUND((unit_price * quantity)::numeric, 2)
WHERE subtotal = 0;

UPDATE orders.orders o
SET total_amount = sub.tot
FROM (
    SELECT order_id, ROUND(SUM(subtotal)::numeric, 2) AS tot
    FROM orders.order_items GROUP BY order_id
) sub
WHERE o.id = sub.order_id;

DELETE FROM analytics.sales_daily;
INSERT INTO analytics.sales_daily (id, date, total_orders, total_revenue, avg_order_size, created_at, updated_at)
SELECT gen_random_uuid(),
       date_trunc('day', created_at)::date,
       COUNT(*)::int,
       ROUND(SUM(total_amount)::numeric, 2),
       ROUND(AVG(total_amount)::numeric, 2),
       NOW(), NOW()
FROM orders.orders
WHERE status NOT IN ('cancelled', 'returned')
GROUP BY 2;

COMMIT;
