
BEGIN;

UPDATE orders.order_items
SET quantity = CASE
        WHEN quantity > 50 THEN GREATEST(5, (quantity / 10)::int)
        WHEN quantity > 25 THEN GREATEST(5, (quantity / 2)::int)
        ELSE quantity
    END
WHERE quantity > 25;

UPDATE orders.order_items
SET subtotal = ROUND((unit_price * quantity)::numeric, 2);

UPDATE orders.orders o
SET total_amount = sub.tot
FROM (
    SELECT order_id, ROUND(SUM(subtotal)::numeric, 2) AS tot
    FROM orders.order_items GROUP BY order_id
) sub
WHERE o.id = sub.order_id;

WITH active_carriers AS (
    SELECT id, row_number() OVER (ORDER BY created_at) AS rn,
           count(*) OVER () AS total
    FROM logistics.carrier WHERE is_active = true
),
all_warehouses AS (
    SELECT id, row_number() OVER (ORDER BY created_at) AS rn,
           count(*) OVER () AS total
    FROM inventory.warehouse WHERE is_active = true
)
UPDATE logistics.shipment s
SET carrier_id = (
        SELECT id FROM active_carriers
        WHERE rn = (abs(hashtextextended(s.id::text, 21) % total) + 1)
    ),
    warehouse_id = (
        SELECT id FROM all_warehouses
        WHERE rn = (abs(hashtextextended(s.id::text, 22) % total) + 1)
    )
WHERE s.created_at < NOW() - interval '7 days';

WITH customers AS (
    SELECT customer_name FROM orders.orders
    GROUP BY customer_name ORDER BY count(*) DESC LIMIT 60
),
synthetic_customers AS (
    SELECT first || ' ' || last AS customer_name
    FROM (VALUES
        ('Yulian',    'Marusiak'),     ('Bohdana',   'Pyliuk'),
        ('Solomiia',  'Hrabovska'),    ('Sviatoslav','Drobotia'),
        ('Bogdana',   'Zelenchuk'),    ('Lubomyr',   'Shevchyk'),
        ('Khrystyna', 'Berezovska'),   ('Mykhailo',  'Burlachenko'),
        ('Mariia',    'Yankovska'),    ('Dmytro',    'Ostapchuk'),
        ('Olesia',    'Lukianenko'),   ('Petro',     'Kuzmenko'),
        ('Zoryana',   'Hradova'),      ('Andriana',  'Dovbenko'),
        ('Vasyl',     'Mohyla'),       ('Yana',      'Mykhailenko'),
        ('Maksym',    'Cherniavskyi'), ('Roksoliana','Sushko'),
        ('Borys',     'Hnatchuk'),     ('Olha',      'Polishchuk'),
        ('Stepan',    'Khoma'),        ('Marichka',  'Khomenko'),
        ('Lukian',    'Ostapenko'),    ('Karina',    'Reshetnyk'),
        ('Yarema',    'Mazepa'),       ('Iryna',     'Zaverukha'),
        ('Pavlo',     'Domanskyi'),    ('Lesia',     'Holub'),
        ('Khyma',     'Kovalska'),     ('Anatolii',  'Pylypchuk')
    ) v(first, last)
),
all_customers AS (
    SELECT customer_name FROM customers
    UNION ALL
    SELECT customer_name FROM synthetic_customers
),
product_pool AS (
    SELECT id::text AS pid, name, unit_price, category, row_number() OVER () AS rn,
           count(*) OVER () AS total
    FROM inventory.product
),
inserted_orders AS (
    INSERT INTO orders.orders (id, customer_name, status, total_amount, created_at, updated_at)
    SELECT
        gen_random_uuid(),
        (SELECT customer_name FROM all_customers
         ORDER BY md5(day::text || idx::text || '7') LIMIT 1),
        CASE
            WHEN (abs(hashtextextended(day::text || idx::text, 31) % 100)) < 5 THEN 'cancelled'
            WHEN day < NOW() - interval '30 days' THEN 'completed'
            WHEN (abs(hashtextextended(day::text || idx::text, 32) % 100)) < 80 THEN 'delivered'
            ELSE 'shipped'
        END::varchar,
        0,
        day - (random() * interval '14 hours'),
        day - (random() * interval '14 hours')
    FROM (
        SELECT gs.day, idx
        FROM generate_series(
            date_trunc('day', NOW() - interval '23 months'),
            date_trunc('day', NOW() - interval '12 months'),
            interval '1 day'
        ) gs(day)
        CROSS JOIN LATERAL generate_series(1, 6 + (abs(hashtextextended(gs.day::text, 33) % 8))::int) idx
    ) d(day, idx)
    RETURNING id, created_at
),
new_items AS (
    INSERT INTO orders.order_items (id, order_id, product_id, name, quantity, unit_price, subtotal)
    SELECT
        gen_random_uuid(),
        io.id,
        pp.pid,
        pp.name,
        (1 + (abs(hashtextextended(io.id::text || line.idx::text, 41) % 12)))::int AS qty,
        ROUND((pp.unit_price * (0.85 + (abs(hashtextextended(io.id::text || line.idx::text, 42) % 1000)::numeric / 1000.0) * 0.30))::numeric, 2) AS up,
        0::numeric
    FROM inserted_orders io
    CROSS JOIN generate_series(1, 1 + (abs(hashtextextended(io.id::text, 43) % 4))::int) line(idx)
    CROSS JOIN LATERAL (
        SELECT * FROM product_pool
        WHERE rn = ((abs(hashtextextended(io.id::text || line.idx::text, 44)) % total) + 1)
    ) pp
    RETURNING order_id, quantity, unit_price
)
SELECT count(*) AS rows_inserted FROM new_items;

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

DELETE FROM analytics.logistics_daily;
INSERT INTO analytics.logistics_daily (id, date, total_shipments, delivered_count, failed_count, avg_delivery_hours, on_time_rate, created_at, updated_at)
SELECT gen_random_uuid(),
       date_trunc('day', created_at)::date,
       COUNT(*)::int,
       COUNT(*) FILTER (WHERE status = 'delivered')::int,
       COUNT(*) FILTER (WHERE status IN ('failed', 'returned', 'returned_to_sender'))::int,
       COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (delivered_at - created_at)) / 3600.0)::numeric, 2), 0),
       CASE WHEN COUNT(*) > 0 THEN
            ROUND((100.0 * COUNT(*) FILTER (WHERE status = 'delivered') / COUNT(*))::numeric, 2)
            ELSE 0 END,
       NOW(), NOW()
FROM logistics.shipment
GROUP BY 2;

DELETE FROM analytics.inventory_snapshot;
INSERT INTO analytics.inventory_snapshot (id, date, total_products, total_quantity, total_reserved, total_available, low_stock_count, created_at, updated_at)
SELECT gen_random_uuid(),
       (NOW() - (gs * interval '1 day'))::date,
       COUNT(DISTINCT product_id)::int,
       SUM(quantity)::int,
       SUM(reserved)::int,
       SUM(quantity - reserved)::int,
       COUNT(*) FILTER (WHERE quantity < min_threshold)::int,
       NOW(), NOW()
FROM inventory.stock, generate_series(0, 90) gs
GROUP BY gs;

COMMIT;
