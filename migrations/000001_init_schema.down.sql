-- migration_down.sql
BEGIN;

-- Удаление триггеров
DROP TRIGGER IF EXISTS product_insert_trigger ON product;
DROP TRIGGER IF EXISTS reception_status_update ON reception;

-- Удаление функций
DROP FUNCTION IF EXISTS update_reception_item_count();
DROP FUNCTION IF EXISTS update_reception_close_time();

-- Удаление таблиц в порядке, соблюдающем ограничения внешних ключей
DROP TABLE IF EXISTS user_pvz;
DROP TABLE IF EXISTS product;
DROP TABLE IF EXISTS reception;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS pvz;

COMMIT;
