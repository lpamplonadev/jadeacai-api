CREATE TABLE IF NOT EXISTS catalog_gourmet (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gourmet_key text NOT NULL UNIQUE,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    tag text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    image_alt text NOT NULL DEFAULT '',
    available boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_gourmet_items (
    gourmet_id uuid NOT NULL REFERENCES catalog_gourmet(id) ON DELETE CASCADE,
    item_id uuid NOT NULL REFERENCES catalog_items(id) ON DELETE RESTRICT,
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (gourmet_id, item_id)
);

CREATE TABLE IF NOT EXISTS catalog_gourmet_sizes (
    gourmet_id uuid NOT NULL REFERENCES catalog_gourmet(id) ON DELETE CASCADE,
    size_item_id uuid NOT NULL REFERENCES catalog_items(id) ON DELETE RESTRICT,
    price_cents integer NOT NULL CHECK (price_cents >= 0),
    sort_order integer NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (gourmet_id, size_item_id)
);

CREATE INDEX IF NOT EXISTS catalog_gourmet_available_sort_idx
    ON catalog_gourmet (sort_order, name)
    WHERE available AND deleted_at IS NULL;

INSERT INTO catalog_gourmet (
    id, gourmet_key, name, description, tag, image_url, image_alt,
    available, sort_order, deleted_at, created_at, updated_at
)
SELECT id, combo_key, name, description, tag, image_url, image_alt,
    available, sort_order, deleted_at, created_at, updated_at
FROM catalog_combos
WHERE store_category = 'gourmet'
ON CONFLICT (id) DO NOTHING;

INSERT INTO catalog_gourmet_items (gourmet_id, item_id, quantity, sort_order)
SELECT relation.combo_id, relation.item_id, relation.quantity, row_number() OVER (
    PARTITION BY relation.combo_id ORDER BY item.kind, item.sort_order, item.name
) - 1
FROM catalog_combo_items AS relation
JOIN catalog_combos AS combo ON combo.id = relation.combo_id
JOIN catalog_items AS item ON item.id = relation.item_id
WHERE combo.store_category = 'gourmet'
    AND item.kind <> 'size'
ON CONFLICT (gourmet_id, item_id) DO NOTHING;

INSERT INTO catalog_gourmet_sizes (gourmet_id, size_item_id, price_cents, sort_order)
SELECT variants.combo_id, variants.size_item_id, variants.price_cents, variants.sort_order
FROM catalog_combo_gourmet_sizes AS variants
JOIN catalog_combos AS combo ON combo.id = variants.combo_id
WHERE combo.store_category = 'gourmet'
ON CONFLICT (gourmet_id, size_item_id) DO NOTHING;

INSERT INTO catalog_gourmet_sizes (gourmet_id, size_item_id, price_cents, sort_order)
SELECT combo.id, combo.size_item_id, combo.price_cents, 0
FROM catalog_combos AS combo
WHERE combo.store_category = 'gourmet'
  AND NOT EXISTS (
      SELECT 1 FROM catalog_gourmet_sizes AS variants
      WHERE variants.gourmet_id = combo.id
  )
ON CONFLICT (gourmet_id, size_item_id) DO NOTHING;

DELETE FROM catalog_combo_items
WHERE combo_id IN (SELECT id FROM catalog_combos WHERE store_category = 'gourmet');

DELETE FROM catalog_combo_gourmet_sizes
WHERE combo_id IN (SELECT id FROM catalog_combos WHERE store_category = 'gourmet');

DELETE FROM catalog_combos WHERE store_category = 'gourmet';
DROP TABLE IF EXISTS catalog_combo_gourmet_sizes;

ALTER TABLE catalog_combos
    DROP COLUMN IF EXISTS store_category,
    DROP COLUMN IF EXISTS description;

ALTER TABLE catalog_gourmet ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_gourmet_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_gourmet_sizes ENABLE ROW LEVEL SECURITY;