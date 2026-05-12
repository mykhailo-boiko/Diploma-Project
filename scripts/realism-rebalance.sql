
BEGIN;

CREATE TEMP TABLE _band(category text PRIMARY KEY, min_p numeric, max_p numeric);

INSERT INTO _band(category, min_p, max_p) VALUES
    ('Electronics',     15,    2500),
    ('Furniture',       80,    1500),
    ('Clothing',        15,     200),
    ('Food',             3,      50),
    ('Food & Beverage',  5,      40),
    ('Tools',           10,     500),
    ('Industrial',     200,    5000),
    ('Office',           3,     300),
    ('Sports',          20,     500),
    ('Automotive',      30,    2000),
    ('Health',           5,     200),
    ('Other',           10,     200);

UPDATE inventory.product p
SET unit_price = ROUND(
        (b.min_p + (abs(hashtextextended(p.sku, 7) % 10000)::numeric / 10000.0)
         * (b.max_p - b.min_p))::numeric,
        2),
    updated_at = NOW()
FROM _band b
WHERE p.category = b.category;

UPDATE inventory.product SET unit_price = LEAST(unit_price, 200)
WHERE category NOT IN (SELECT category FROM _band);

UPDATE inventory.product p
SET unit_price = ROUND((p.unit_price * 0.30)::numeric, 2)
FROM _band b
WHERE p.category = b.category
  AND abs(hashtextextended(p.sku, 11) % 100) >= 25;

UPDATE inventory.product SET unit_price = GREATEST(unit_price, 1.99);

UPDATE orders.order_items oi
SET unit_price = ROUND((p.unit_price *
        (0.85 + (abs(hashtextextended(oi.id::text, 13) % 1000)::numeric / 1000.0) * 0.30))::numeric, 2),
    subtotal = ROUND((p.unit_price *
        (0.85 + (abs(hashtextextended(oi.id::text, 13) % 1000)::numeric / 1000.0) * 0.30) *
        oi.quantity)::numeric, 2)
FROM inventory.product p
WHERE oi.product_id = p.id::text;

UPDATE orders.orders o
SET total_amount = sub.tot,
    updated_at = COALESCE(o.updated_at, NOW())
FROM (
    SELECT order_id, ROUND(SUM(subtotal)::numeric, 2) AS tot
    FROM orders.order_items GROUP BY order_id
) sub
WHERE o.id = sub.order_id;

INSERT INTO inventory.warehouse (id, name, address, is_active, created_at, updated_at)
SELECT gen_random_uuid(), n.name, n.addr, true,
       NOW() - (random() * interval '300 days'), NOW()
FROM (VALUES
    ('Kharkiv Industrial Park',    'Nauky Ave, 87, Kharkiv, Ukraine 61000'),
    ('Dnipro South Distribution',  'Slobozhanskyi Ave, 119, Dnipro, Ukraine 49000'),
    ('Vinnytsia West Hub',         'Khmelnytske Shose, 23, Vinnytsia, Ukraine 21000'),
    ('Zaporizhzhia Steel District','Sobornyi Ave, 165, Zaporizhzhia, Ukraine 69000'),
    ('Ivano-Frankivsk Regional',   'Halytska St, 50, Ivano-Frankivsk, Ukraine 76000'),
    ('Poltava Forwarding Center',  'Sobornosti St, 76, Poltava, Ukraine 36000'),
    ('Cherkasy Riverside Storage', 'Khreshchatyk St, 200, Cherkasy, Ukraine 18000'),
    ('Mykolaiv Port Warehouse',    'Heroiv Stalinhrada Ave, 1, Mykolaiv, Ukraine 54000'),
    ('Chernivtsi Border Depot',    'Holovna St, 145, Chernivtsi, Ukraine 58000'),
    ('Uzhhorod Cross-Border Hub',  'Sobranetska St, 36, Uzhhorod, Ukraine 88000'),
    ('Warsaw Logistics Park',      'ul. Polna 52, Warsaw, Poland 00-644'),
    ('Berlin Express Hub',         'Friedrichstrasse 142, Berlin, Germany 10117')
) n(name, addr)
WHERE NOT EXISTS (SELECT 1 FROM inventory.warehouse w WHERE w.name = n.name);

ALTER TABLE logistics.carrier DROP CONSTRAINT IF EXISTS carrier_type_check;
ALTER TABLE logistics.carrier ADD CONSTRAINT carrier_type_check
    CHECK (type IN ('ground', 'air', 'sea', 'rail', 'express'));

INSERT INTO logistics.carrier (id, name, type, cost_per_km, is_active, created_at, updated_at)
SELECT gen_random_uuid(), c.name, c.typ, c.cost, c.active,
       NOW() - (random() * interval '600 days'), NOW()
FROM (VALUES
    ('NovaPoshta Express',     'ground',   1.85, true),
    ('UkrPost National',       'ground',   1.10, true),
    ('Meest Express',          'ground',   2.05, true),
    ('Justin Logistics',       'ground',   2.30, true),
    ('DHL Europe',             'express',  5.95, true),
    ('FedEx Air',              'air',      8.10, true),
    ('UPS Worldwide',          'express',  5.40, true),
    ('DPD Network',            'express',  3.80, true),
    ('Ralwex Cargo',           'rail',     0.95, true),
    ('BlueWater Maritime',     'sea',      0.85, true),
    ('DniproRail Freight',     'rail',     1.05, true),
    ('AirCharter Special',     'air',     11.50, true),
    ('Continental Cargo',      'ground',   2.20, true),
    ('AlpineFreight Express',  'express',  4.75, true),
    ('BalticChannel Shipping', 'sea',      1.15, true),
    ('EuropExpress 24h',       'express',  6.80, false),
    ('LocalFleet Last-Mile',   'ground',   3.40, true)
) c(name, typ, cost, active)
WHERE NOT EXISTS (SELECT 1 FROM logistics.carrier x WHERE x.name = c.name);

WITH admin_hash AS (
    SELECT password_hash FROM users.users WHERE email = 'admin@chainorchestra.local'
)
INSERT INTO users.users (id, email, password_hash, first_name, last_name, role, created_at, updated_at)
SELECT gen_random_uuid(), u.email, (SELECT password_hash FROM admin_hash),
       u.first, u.last, u.role,
       NOW() - (random() * interval '500 days'), NOW()
FROM (VALUES
    ('andrii.melnyk@chainorchestra.local',     'Andrii',    'Melnyk',    'operator'),
    ('bohdan.tkachuk@chainorchestra.local',    'Bohdan',    'Tkachuk',   'operator'),
    ('daryna.savchenko@chainorchestra.local',  'Daryna',    'Savchenko', 'operator'),
    ('halyna.koval@chainorchestra.local',      'Halyna',    'Koval',     'operator'),
    ('ihor.lytvyn@chainorchestra.local',       'Ihor',      'Lytvyn',    'operator'),
    ('kateryna.romanenko@chainorchestra.local','Kateryna',  'Romanenko', 'operator'),
    ('liliia.pavlenko@chainorchestra.local',   'Liliia',    'Pavlenko',  'operator'),
    ('marko.boiko@chainorchestra.local',       'Marko',     'Boiko',     'operator'),
    ('nazar.hrytsenko@chainorchestra.local',   'Nazar',     'Hrytsenko', 'warehouse_manager'),
    ('olha.lysenko@chainorchestra.local',      'Olha',      'Lysenko',   'warehouse_manager'),
    ('petro.marchenko@chainorchestra.local',   'Petro',     'Marchenko', 'warehouse_manager'),
    ('roman.sydorenko@chainorchestra.local',   'Roman',     'Sydorenko', 'warehouse_manager'),
    ('sofiia.voloshyn@chainorchestra.local',   'Sofiia',    'Voloshyn',  'warehouse_manager'),
    ('taras.yermak@chainorchestra.local',      'Taras',     'Yermak',    'warehouse_manager'),
    ('vira.bondarenko@chainorchestra.local',   'Vira',      'Bondarenko','logistics_manager'),
    ('yaroslav.kovalenko@chainorchestra.local','Yaroslav',  'Kovalenko', 'logistics_manager'),
    ('zoryana.shevchenko@chainorchestra.local','Zoryana',   'Shevchenko','logistics_manager'),
    ('anton.morozenko@chainorchestra.local',   'Anton',     'Morozenko', 'logistics_manager'),
    ('valeriia.petrenko@chainorchestra.local', 'Valeriia',  'Petrenko',  'logistics_manager'),
    ('mykola.kravchuk@chainorchestra.local',   'Mykola',    'Kravchuk',  'logistics_manager'),
    ('iryna.osadcha@chainorchestra.local',     'Iryna',     'Osadcha',   'analyst'),
    ('vitalii.zhuk@chainorchestra.local',      'Vitalii',   'Zhuk',      'analyst'),
    ('yuliia.tarasenko@chainorchestra.local',  'Yuliia',    'Tarasenko', 'analyst'),
    ('serhii.honchar@chainorchestra.local',    'Serhii',    'Honchar',   'analyst'),
    ('oleksandr.bilenko@chainorchestra.local', 'Oleksandr', 'Bilenko',   'admin'),
    ('mariana.kolesnyk@chainorchestra.local',  'Mariana',   'Kolesnyk',  'admin'),
    ('dmytro.svitlychnyi@chainorchestra.local','Dmytro',    'Svitlychnyi','operator'),
    ('alina.fedoriv@chainorchestra.local',     'Alina',     'Fedoriv',   'operator'),
    ('rostyslav.bilous@chainorchestra.local',  'Rostyslav', 'Bilous',    'warehouse_manager'),
    ('olesia.sokolova@chainorchestra.local',   'Olesia',    'Sokolova',  'analyst')
) u(email, first, last, role)
WHERE NOT EXISTS (SELECT 1 FROM users.users x WHERE x.email = u.email);

INSERT INTO inventory.stock (id, product_id, warehouse_id, quantity, reserved, min_threshold, updated_at)
SELECT gen_random_uuid(),
       p.id, w.id,
       (50 + (abs(hashtextextended(p.id::text || w.id::text, 1) % 950)))::int,
       (abs(hashtextextended(p.id::text || w.id::text, 2) % 30))::int,
       (10 + (abs(hashtextextended(p.id::text || w.id::text, 3) % 40)))::int,
       NOW() - (random() * interval '90 days')
FROM inventory.product p
CROSS JOIN inventory.warehouse w
WHERE NOT EXISTS (
    SELECT 1 FROM inventory.stock s
    WHERE s.product_id = p.id AND s.warehouse_id = w.id
);

UPDATE inventory.stock
SET quantity = LEAST(min_threshold - 1, GREATEST(0, quantity))
WHERE id IN (
    SELECT id FROM inventory.stock
    WHERE abs(hashtextextended(id::text, 4) % 100) < 8
);

WITH targets AS (
    SELECT s.id AS stock_id, s.product_id, s.warehouse_id
    FROM inventory.stock s
    ORDER BY random()
    LIMIT 2500
)
INSERT INTO inventory.stock_movement (id, stock_id, product_id, warehouse_id, type, quantity, reference, created_at)
SELECT
    gen_random_uuid(),
    t.stock_id, t.product_id, t.warehouse_id,
    CASE WHEN r.bucket < 55 THEN 'inbound'
         WHEN r.bucket < 90 THEN 'outbound'
         ELSE 'adjustment' END AS type,
    CASE WHEN r.bucket < 55 THEN (5 + (abs(hashtextextended(t.stock_id::text, r.seed) % 80)))::int
         WHEN r.bucket < 90 THEN -(1 + (abs(hashtextextended(t.stock_id::text, r.seed) % 25)))::int
         ELSE ((abs(hashtextextended(t.stock_id::text, r.seed) % 41)) - 20)::int END AS qty,
    'historical-' || r.seed::text,
    NOW() - (random() * interval '365 days')
FROM targets t
CROSS JOIN LATERAL (
    SELECT (abs(hashtextextended(t.stock_id::text, gs) % 100))::int AS bucket, gs AS seed
    FROM generate_series(1, 1) gs
) r;

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

INSERT INTO notifications.notification (id, user_id, type, title, message, status, created_at)
SELECT gen_random_uuid(), u.id,
       'low_stock',
       'Low stock — ' || p.name,
       'Warehouse ' || w.name || ' is below threshold for ' || p.name || ' (only '
        || s.quantity || ' units left, threshold ' || s.min_threshold || ')',
       'unread',
       NOW() - (random() * interval '15 days')
FROM inventory.stock s
JOIN inventory.product p ON p.id = s.product_id
JOIN inventory.warehouse w ON w.id = s.warehouse_id
CROSS JOIN LATERAL (
    SELECT id FROM users.users
    WHERE role IN ('warehouse_manager', 'admin')
    ORDER BY random() LIMIT 1
) u
WHERE s.quantity < s.min_threshold
  AND random() < 0.40
LIMIT 200;

COMMIT;
