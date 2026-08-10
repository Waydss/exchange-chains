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

CREATE TABLE user_shadow_registry (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    amount INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_shadow_registry_reason_not_empty
        CHECK (LENGTH(BTRIM(reason)) > 0),
    CONSTRAINT user_shadow_registry_amount_positive
        CHECK (amount > 0)
);

CREATE INDEX user_shadow_registry_user_id_idx ON user_shadow_registry (user_id);
CREATE INDEX user_shadow_registry_created_at_idx ON user_shadow_registry (created_at);

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
        CHECK (status IN ('searching', 'in_chain', 'reserved', 'in_transit', 'exchanged', 'archived'))
);

CREATE INDEX items_user_id_idx ON items (user_id);
CREATE INDEX items_category_id_idx ON items (category_id);
CREATE INDEX items_wanted_category_id_idx ON items (wanted_category_id);
CREATE INDEX items_status_idx ON items (status);

CREATE TABLE item_photos (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX item_photos_item_id_idx ON item_photos (item_id);

CREATE TABLE wishes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    price_min BIGINT NOT NULL,
    price_max BIGINT NOT NULL,
    title_keywords TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT wishes_price_min_positive
        CHECK (price_min > 0),
    CONSTRAINT wishes_price_max_positive
        CHECK (price_max > 0),
    CONSTRAINT wishes_price_range_valid
        CHECK (price_min <= price_max)
);

CREATE INDEX wishes_user_id_idx ON wishes (user_id);
CREATE INDEX wishes_category_id_idx ON wishes (category_id);
CREATE INDEX wishes_category_price_idx ON wishes (category_id, price_min, price_max);

CREATE TABLE chains (
    id BIGSERIAL PRIMARY KEY,
    root_user_id BIGINT NOT NULL REFERENCES users(id),
    root_wish_id BIGINT NOT NULL REFERENCES wishes(id),
    signature TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'matched',
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    locked_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancelled_by_user_id BIGINT REFERENCES users(id),
    cancel_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chains_signature_not_empty
        CHECK (LENGTH(BTRIM(signature)) > 0),
    CONSTRAINT chains_status_valid
        CHECK (status IN ('matched', 'pending', 'locked', 'in_delivery', 'returning', 'completed', 'cancelled', 'disputed')),
    CONSTRAINT chains_score_non_negative
        CHECK (score >= 0)
);

CREATE INDEX chains_root_user_id_idx ON chains (root_user_id);
CREATE INDEX chains_root_wish_id_idx ON chains (root_wish_id);
CREATE INDEX chains_status_score_idx ON chains (status, score DESC);
CREATE INDEX chains_expires_at_idx ON chains (expires_at);

CREATE TABLE chain_participants (
    chain_id BIGINT NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending',
    accepted_at TIMESTAMPTZ,
    declined_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chain_participants_pk
        PRIMARY KEY (chain_id, user_id),
    CONSTRAINT chain_participants_status_valid
        CHECK (status IN ('pending', 'accepted', 'declined', 'expired'))
);

CREATE INDEX chain_participants_user_id_idx ON chain_participants (user_id);
CREATE INDEX chain_participants_status_idx ON chain_participants (status);

CREATE TABLE chain_barters (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    from_user_id BIGINT NOT NULL REFERENCES users(id),
    to_user_id BIGINT NOT NULL REFERENCES users(id),
    give_item_id BIGINT NOT NULL REFERENCES items(id),
    wish_id BIGINT NOT NULL REFERENCES wishes(id),
    position INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    transfer_method TEXT,
    given_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    return_started_at TIMESTAMPTZ,
    returned_at TIMESTAMPTZ,
    rejection_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chain_barters_position_non_negative
        CHECK (position >= 0),
    CONSTRAINT chain_barters_users_differ
        CHECK (from_user_id <> to_user_id),
    CONSTRAINT chain_barters_status_valid
        CHECK (status IN ('pending', 'method_selected', 'given', 'received', 'rejected', 'returning', 'returned')),
    CONSTRAINT chain_barters_transfer_method_valid
        CHECK (transfer_method IS NULL OR transfer_method IN ('meetup', 'delivery')),
    CONSTRAINT chain_barters_position_unique
        UNIQUE (chain_id, position)
);

CREATE INDEX chain_barters_chain_id_idx ON chain_barters (chain_id);
CREATE INDEX chain_barters_from_user_id_idx ON chain_barters (from_user_id);
CREATE INDEX chain_barters_to_user_id_idx ON chain_barters (to_user_id);
CREATE INDEX chain_barters_give_item_id_idx ON chain_barters (give_item_id);
CREATE INDEX chain_barters_wish_id_idx ON chain_barters (wish_id);
CREATE INDEX chain_barters_status_idx ON chain_barters (status);

CREATE TABLE user_ratings (
    id BIGSERIAL PRIMARY KEY,
    from_id BIGINT NOT NULL REFERENCES users(id),
    to_id BIGINT NOT NULL REFERENCES users(id),
    barter_id BIGINT NOT NULL REFERENCES chain_barters(id),
    public_rating NUMERIC(3, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_ratings_users_differ
        CHECK (from_id <> to_id),
    CONSTRAINT user_ratings_public_rating_valid
        CHECK (public_rating >= 0 AND public_rating <= 5),
    CONSTRAINT user_ratings_once_per_barter
        UNIQUE (from_id, to_id, barter_id)
);

CREATE INDEX user_ratings_to_id_idx ON user_ratings (to_id);
CREATE INDEX user_ratings_from_id_idx ON user_ratings (from_id);
CREATE INDEX user_ratings_barter_id_idx ON user_ratings (barter_id);
