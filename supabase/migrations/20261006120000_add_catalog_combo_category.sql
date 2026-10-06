ALTER TABLE catalog_combos
    ADD COLUMN IF NOT EXISTS store_category text NOT NULL DEFAULT 'combo'
    CHECK (store_category IN ('combo', 'gourmet'));