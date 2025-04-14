-- migration_up.sql
BEGIN;

-- Создание таблицы ПВЗ
CREATE TABLE pvz (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    city VARCHAR(100) NOT NULL,
    address TEXT NOT NULL,
    registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    working_hours VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance'))
);

CREATE INDEX idx_pvz_city ON pvz(city);
CREATE INDEX idx_pvz_status ON pvz(status);

-- Создание таблицы пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('employee', 'moderator')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

-- Создание таблицы приёмки товара
CREATE TABLE reception (
    id SERIAL PRIMARY KEY,
    datetime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pvz_id INTEGER NOT NULL REFERENCES pvz(id),
    moderator_id INTEGER REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'close')),
    comment TEXT,
    close_datetime TIMESTAMP,
    total_items INTEGER DEFAULT 0
);

CREATE INDEX idx_reception_status ON reception(status);
CREATE INDEX idx_reception_pvz_id ON reception(pvz_id);
CREATE INDEX idx_reception_moderator_id ON reception(moderator_id);

-- Создание таблицы товаров
CREATE TABLE product (
    id SERIAL PRIMARY KEY,
    reception_id INTEGER NOT NULL REFERENCES reception(id),
    reception_datetime TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type VARCHAR(20) NOT NULL CHECK (type IN ('electronics', 'clothes', 'shoes')),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    barcode VARCHAR(50) UNIQUE,
    status VARCHAR(20) DEFAULT 'available' CHECK (status IN ('available', 'sold', 'damaged', 'reserved'))
);

CREATE INDEX idx_product_reception_id ON product(reception_id);
CREATE INDEX idx_product_type ON product(type);
CREATE INDEX idx_product_status ON product(status);
CREATE INDEX idx_product_barcode ON product(barcode);

-- Создание связующей таблицы пользователей и ПВЗ
CREATE TABLE user_pvz (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    pvz_id INTEGER NOT NULL REFERENCES pvz(id),
    role VARCHAR(20) NOT NULL CHECK (role IN ('manager', 'employee', 'moderator')),
    assigned_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    UNIQUE(user_id, pvz_id)
);

CREATE INDEX idx_user_pvz_user_id ON user_pvz(user_id);
CREATE INDEX idx_user_pvz_pvz_id ON user_pvz(pvz_id);

-- Триггер для обновления времени закрытия приёмки
CREATE OR REPLACE FUNCTION update_reception_close_time()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'close' AND OLD.status = 'in_progress' THEN
        NEW.close_datetime = CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER reception_status_update
BEFORE UPDATE ON reception
FOR EACH ROW
EXECUTE FUNCTION update_reception_close_time();

-- Триггер для подсчета товаров в приёмке
CREATE OR REPLACE FUNCTION update_reception_item_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE reception 
    SET total_items = total_items + 1
    WHERE id = NEW.reception_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER product_insert_trigger
AFTER INSERT ON product
FOR EACH ROW
EXECUTE FUNCTION update_reception_item_count();

COMMIT;
