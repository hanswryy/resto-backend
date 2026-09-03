CREATE EXTENSION IF NOT EXISTS pgcrypto;

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS users;

-- =========================================================
-- USERS
-- =========================================================
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    device_token  TEXT,                       -- token FCM/Expo, di-update dari app
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================================================
-- MENU ITEMS
-- =========================================================
CREATE TABLE menu_items (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price       INTEGER NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================================================
-- ORDERS  (header pesanan)
-- =========================================================
CREATE TABLE orders (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    status     TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','preparing','ready','completed','cancelled')),
    total      INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================================================
-- ORDER ITEMS  (baris detail tiap pesanan)
-- =========================================================
CREATE TABLE order_items (
    id             BIGSERIAL PRIMARY KEY,
    order_id       BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id   BIGINT NOT NULL REFERENCES menu_items(id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    price_at_order INTEGER NOT NULL
);

-- =========================================================
-- SEED DATA
-- =========================================================
-- Password kedua user: "password123"
INSERT INTO users (email, password_hash) VALUES
    ('customer@resto.test', crypt('password123', gen_salt('bf'))),
    ('staff@resto.test',    crypt('password123', gen_salt('bf')));

INSERT INTO menu_items (name, description, price) VALUES
    ('Nasi Goreng Spesial', 'Nasi goreng dengan telur, ayam, dan kerupuk', 25000),
    ('Mie Ayam Bakso',      'Mie ayam dengan bakso sapi dan pangsit',      22000),
    ('Es Teh Manis',        'Teh manis dingin segar',                       5000),
    ('Ayam Bakar Madu',     'Ayam bakar bumbu madu dengan lalapan',        30000),
    ('Jus Alpukat',         'Jus alpukat segar dengan susu cokelat',       15000);