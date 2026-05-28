-- +goose Up

ALTER TABLE certifications
  ADD COLUMN category TEXT NOT NULL DEFAULT 'raw'
    CHECK (category IN ('raw','psa9','psa10','cgc9','cgc10')),
  ADD COLUMN purpose  TEXT NOT NULL DEFAULT 'analytics'
    CHECK (purpose IN ('analytics','buy_and_grade','crack_and_regrade'));

ALTER TABLE cards
  ADD COLUMN image_url          TEXT,
  ADD COLUMN market_display_key TEXT;

-- +goose Down

ALTER TABLE certifications
  DROP COLUMN category,
  DROP COLUMN purpose;

ALTER TABLE cards
  DROP COLUMN image_url,
  DROP COLUMN market_display_key;
