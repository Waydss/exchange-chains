CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    public_rating NUMERIC(3, 2) NOT NULL DEFAULT 0,
    shadow_rating INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_login_not_empty
        CHECK (LENGTH(TRIM(login)) > 0)
);
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT categories_name_not_empty
        CHECK (LENGTH(BTRIM(name)) > 0)
);

CREATE UNIQUE INDEX categories_name_unique_idx
    ON categories (LOWER(name));

-- MVP category seed. Keep this list easy to find and edit as the catalogue evolves.
INSERT INTO categories (name) VALUES
    ('Гитары'),
    ('Пианино'),
    ('Палатки'),
    ('Барабаны'),
    ('Клавишные'),
    ('Автомобили'),
    ('Гаражи');

CREATE TABLE items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    condition TEXT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    price_min BIGINT NOT NULL,
    price_max BIGINT NOT NULL,
    wanted_category_id BIGINT NOT NULL REFERENCES categories(id),
    status TEXT NOT NULL DEFAULT 'searching',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT items_title_not_empty
        CHECK (LENGTH(BTRIM(title)) > 0),
    CONSTRAINT items_price_min_positive
        CHECK (price_min > 0),
    CONSTRAINT items_price_max_positive
        CHECK (price_max > 0),
    CONSTRAINT items_price_range_valid
        CHECK (price_min <= price_max),
    CONSTRAINT items_condition_valid
        CHECK (condition IN ('new', 'like_new', 'good', 'fair', 'poor')),
    CONSTRAINT items_status_valid
        CHECK (status IN ('searching', 'in_chain', 'exchanged', 'archived'))
);

CREATE INDEX items_user_id_idx ON items (user_id);
CREATE INDEX items_category_id_idx ON items (category_id);
CREATE INDEX items_wanted_category_id_idx ON items (wanted_category_id);
CREATE INDEX items_status_idx ON items (status);

CREATE TABLE item_photos (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX item_photos_item_id_idx ON item_photos (item_id);
