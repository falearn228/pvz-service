BEGIN;

-- Создание таблицы ПВЗ
CREATE TABLE pvz (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city VARCHAR(100) NOT NULL CHECK (city IN ('Москва', 'Санкт-Петербург', 'Казань')),
    registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pvz_city ON pvz(city);

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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datetime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pvz_id UUID NOT NULL REFERENCES pvz(id),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'close'))
);

CREATE INDEX idx_reception_status ON reception(status);
CREATE INDEX idx_reception_pvz_id ON reception(pvz_id);

-- Создание таблицы товаров
CREATE TABLE product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reception_id UUID NOT NULL REFERENCES reception(id),
    datetime TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type VARCHAR(20) NOT NULL CHECK (type IN ('электроника', 'одежда', 'обувь'))
);

CREATE INDEX idx_product_reception_id ON product(reception_id);
CREATE INDEX idx_product_type ON product(type);


COMMIT;
