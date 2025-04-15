-- migration_down.sql
BEGIN;

-- Удаление таблиц в порядке, соблюдающем ограничения внешних ключей
DROP TABLE IF EXISTS product;
DROP TABLE IF EXISTS reception;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS pvz;

COMMIT;
