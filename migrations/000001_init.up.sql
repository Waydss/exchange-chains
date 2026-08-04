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
