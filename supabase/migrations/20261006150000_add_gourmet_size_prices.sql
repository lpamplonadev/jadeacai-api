CREATE TABLE IF NOT EXISTS catalog_combo_gourmet_sizes (
    combo_id uuid NOT NULL REFERENCES catalog_combos(id) ON DELETE CASCADE,
    size_item_id uuid NOT NULL REFERENCES catalog_items(id) ON DELETE RESTRICT,
    price_cents integer NOT NULL CHECK (price_cents >= 0),
    sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (combo_id, size_item_id)
);

INSERT INTO catalog_combo_gourmet_sizes (combo_id, size_item_id, price_cents, sort_order)
SELECT id, size_item_id, price_cents, 0
FROM catalog_combos
WHERE store_category = 'gourmet'
ON CONFLICT (combo_id, size_item_id) DO NOTHING;

ALTER TABLE catalog_combo_gourmet_sizes ENABLE ROW LEVEL SECURITY;