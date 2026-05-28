-- +goose Up

CREATE TABLE cards (
    id          BIGSERIAL PRIMARY KEY,
    game        TEXT NOT NULL,
    set_code    TEXT NOT NULL,
    set_name    TEXT NOT NULL,
    card_name   TEXT NOT NULL,
    card_number TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game, set_code, card_number)
);

CREATE TABLE certifications (
    id             BIGSERIAL PRIMARY KEY,
    card_id        BIGINT NOT NULL REFERENCES cards(id),
    cert_number    TEXT NOT NULL UNIQUE,
    grader         TEXT NOT NULL DEFAULT 'PSA',
    grade_received SMALLINT,
    graded_at      DATE,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cert_images (
    id         BIGSERIAL PRIMARY KEY,
    cert_id    BIGINT NOT NULL REFERENCES certifications(id) ON DELETE CASCADE,
    side       TEXT NOT NULL CHECK (side IN ('front', 'back')),
    gcs_path   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cert_id, side)
);

CREATE TABLE inspections (
    id                  BIGSERIAL PRIMARY KEY,
    cert_id             BIGINT NOT NULL REFERENCES certifications(id) ON DELETE CASCADE,

    -- centering: % of border on the dominant side (left/top). 50 = perfect.
    centering_front_lr  SMALLINT CHECK (centering_front_lr BETWEEN 0 AND 100),
    centering_front_tb  SMALLINT CHECK (centering_front_tb BETWEEN 0 AND 100),
    centering_back_lr   SMALLINT CHECK (centering_back_lr BETWEEN 0 AND 100),
    centering_back_tb   SMALLINT CHECK (centering_back_tb BETWEEN 0 AND 100),

    surface_front       TEXT CHECK (surface_front IN ('clean','light_scratch','heavy_scratch','print_line','print_dot')),
    surface_back        TEXT CHECK (surface_back  IN ('clean','light_scratch','heavy_scratch','print_line','print_dot')),

    corner_tl           TEXT CHECK (corner_tl IN ('sharp','light_wear','heavy_wear')),
    corner_tr           TEXT CHECK (corner_tr IN ('sharp','light_wear','heavy_wear')),
    corner_bl           TEXT CHECK (corner_bl IN ('sharp','light_wear','heavy_wear')),
    corner_br           TEXT CHECK (corner_br IN ('sharp','light_wear','heavy_wear')),

    edge_top            TEXT CHECK (edge_top    IN ('clean','light_wear','heavy_wear','nick')),
    edge_bottom         TEXT CHECK (edge_bottom IN ('clean','light_wear','heavy_wear','nick')),
    edge_left           TEXT CHECK (edge_left   IN ('clean','light_wear','heavy_wear','nick')),
    edge_right          TEXT CHECK (edge_right  IN ('clean','light_wear','heavy_wear','nick')),

    notes               TEXT,
    source              TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','auto')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON certifications (card_id);
CREATE INDEX ON inspections (cert_id);

-- +goose Down

DROP TABLE IF EXISTS inspections;
DROP TABLE IF EXISTS cert_images;
DROP TABLE IF EXISTS certifications;
DROP TABLE IF EXISTS cards;
