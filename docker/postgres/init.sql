
CREATE SCHEMA IF NOT EXISTS users;
CREATE SCHEMA IF NOT EXISTS orders;
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS logistics;
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE SCHEMA IF NOT EXISTS notifications;

CREATE USER user_service WITH PASSWORD 'user_service_pass';
CREATE USER order_service WITH PASSWORD 'order_service_pass';
CREATE USER inventory_service WITH PASSWORD 'inventory_service_pass';
CREATE USER logistics_service WITH PASSWORD 'logistics_service_pass';
CREATE USER analytics_service WITH PASSWORD 'analytics_service_pass';
CREATE USER notification_service WITH PASSWORD 'notification_service_pass';

GRANT USAGE ON SCHEMA users TO user_service;
GRANT CREATE ON SCHEMA users TO user_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA users GRANT ALL ON TABLES TO user_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA users GRANT ALL ON SEQUENCES TO user_service;

GRANT USAGE ON SCHEMA orders TO order_service;
GRANT CREATE ON SCHEMA orders TO order_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA orders GRANT ALL ON TABLES TO order_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA orders GRANT ALL ON SEQUENCES TO order_service;

GRANT USAGE ON SCHEMA inventory TO inventory_service;
GRANT CREATE ON SCHEMA inventory TO inventory_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory GRANT ALL ON TABLES TO inventory_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory GRANT ALL ON SEQUENCES TO inventory_service;

GRANT USAGE ON SCHEMA logistics TO logistics_service;
GRANT CREATE ON SCHEMA logistics TO logistics_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA logistics GRANT ALL ON TABLES TO logistics_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA logistics GRANT ALL ON SEQUENCES TO logistics_service;

GRANT USAGE ON SCHEMA analytics TO analytics_service;
GRANT CREATE ON SCHEMA analytics TO analytics_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA analytics GRANT ALL ON TABLES TO analytics_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA analytics GRANT ALL ON SEQUENCES TO analytics_service;

GRANT USAGE ON SCHEMA notifications TO notification_service;
GRANT CREATE ON SCHEMA notifications TO notification_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA notifications GRANT ALL ON TABLES TO notification_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA notifications GRANT ALL ON SEQUENCES TO notification_service;

GRANT USAGE ON SCHEMA users TO analytics_service;
GRANT USAGE ON SCHEMA orders TO analytics_service;
GRANT USAGE ON SCHEMA inventory TO analytics_service;
GRANT USAGE ON SCHEMA logistics TO analytics_service;
ALTER DEFAULT PRIVILEGES FOR USER user_service       IN SCHEMA users     GRANT SELECT ON TABLES TO analytics_service;
ALTER DEFAULT PRIVILEGES FOR USER order_service      IN SCHEMA orders    GRANT SELECT ON TABLES TO analytics_service;
ALTER DEFAULT PRIVILEGES FOR USER inventory_service  IN SCHEMA inventory GRANT SELECT ON TABLES TO analytics_service;
ALTER DEFAULT PRIVILEGES FOR USER logistics_service  IN SCHEMA logistics GRANT SELECT ON TABLES TO analytics_service;
