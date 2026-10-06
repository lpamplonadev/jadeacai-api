ALTER TABLE catalog_combos
    ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';