CREATE TABLE IF NOT EXISTS catalog_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    item_key text NOT NULL UNIQUE,
    kind text NOT NULL CHECK (
        kind IN (
            'flavor',
            'size',
            'topping',
            'sauce',
            'condiment_position',
            'fruit',
            'extra'
        )
    ),
    name text NOT NULL,
    price_cents integer NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    available boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS catalog_items_kind_sort_idx
    ON catalog_items (kind, sort_order, name)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS catalog_combos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    combo_key text NOT NULL UNIQUE,
    name text NOT NULL,
    size_item_id uuid NOT NULL REFERENCES catalog_items(id),
    price_cents integer NOT NULL CHECK (price_cents >= 0),
    included_toppings integer NOT NULL DEFAULT 0 CHECK (included_toppings >= 0),
    included_fruits integer NOT NULL DEFAULT 0 CHECK (included_fruits >= 0),
    included_extras integer NOT NULL DEFAULT 0 CHECK (included_extras >= 0),
    tag text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    image_alt text NOT NULL DEFAULT '',
    available boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS catalog_combos_available_sort_idx
    ON catalog_combos (sort_order, name)
    WHERE available AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS catalog_combo_items (
    combo_id uuid NOT NULL REFERENCES catalog_combos(id) ON DELETE RESTRICT,
    item_id uuid NOT NULL REFERENCES catalog_items(id) ON DELETE RESTRICT,
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (combo_id, item_id)
);

CREATE INDEX IF NOT EXISTS catalog_combo_items_item_idx
    ON catalog_combo_items (item_id);

CREATE TABLE IF NOT EXISTS catalog_rules (
    rule_key text PRIMARY KEY,
    rule_value jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO catalog_items (item_key, kind, name, price_cents, sort_order)
VALUES
    ('flavor-banana', 'flavor', 'Açaí de banana', 0, 1),
    ('flavor-morango', 'flavor', 'Açaí de morango', 0, 2),
    ('size-300', 'size', '300 ml', 1190, 1),
    ('size-500', 'size', '500 ml', 1590, 2),
    ('size-770', 'size', '770 ml', 1790, 3),
    ('size-1000', 'size', 'Marmita', 2590, 4),
    ('topping-canudinho', 'topping', 'Biscoito canudinho', 100, 1),
    ('topping-pacoca', 'topping', 'Paçoca', 100, 2),
    ('topping-granulado', 'topping', 'Granulado', 100, 3),
    ('topping-leite-em-po', 'topping', 'Leite em pó', 100, 4),
    ('topping-disquete', 'topping', 'Disquete (M&M)', 100, 5),
    ('topping-chocoball', 'topping', 'Chocoball', 100, 6),
    ('topping-jujubas', 'topping', 'Jujubas', 100, 7),
    ('topping-amendoim', 'topping', 'Amendoim', 100, 8),
    ('topping-granola', 'topping', 'Granola', 100, 9),
    ('topping-marshmallow', 'topping', 'Marshmallow', 100, 10),
    ('sauce-chocolate', 'sauce', 'Chocolate', 0, 1),
    ('sauce-morango', 'sauce', 'Morango', 0, 2),
    ('sauce-menta', 'sauce', 'Menta', 0, 3),
    ('sauce-uva', 'sauce', 'Uva', 0, 4),
    ('position-bottom', 'condiment_position', 'Embaixo somente', 0, 1),
    ('position-middle', 'condiment_position', 'No meio somente', 0, 2),
    ('position-top', 'condiment_position', 'No topo somente', 0, 3),
    ('position-bottom-middle', 'condiment_position', 'Embaixo e no meio', 0, 4),
    ('position-middle-top', 'condiment_position', 'No meio e no topo', 0, 5),
    ('position-bottom-top', 'condiment_position', 'Embaixo e no topo', 0, 6),
    ('position-all-layers', 'condiment_position', 'Embaixo, no meio e no topo', 300, 7),
    ('fruit-morango', 'fruit', 'Morango', 200, 1),
    ('fruit-banana', 'fruit', 'Banana', 200, 2),
    ('fruit-uva', 'fruit', 'Uva', 200, 3),
    ('extra-leite-condensado', 'extra', 'Leite condensado', 300, 1),
    ('extra-chantilly', 'extra', 'Chantily', 300, 2),
    ('extra-nutella', 'extra', 'Nutella', 300, 3),
    ('extra-ovomaltine', 'extra', 'Ovomaltine', 300, 4)
ON CONFLICT (item_key) DO NOTHING;

INSERT INTO catalog_combos (
    combo_key,
    name,
    size_item_id,
    price_cents,
    included_toppings,
    included_fruits,
    included_extras,
    tag,
    image_url,
    image_alt,
    sort_order
)
SELECT
    seed.combo_key,
    seed.name,
    sizes.id,
    seed.price_cents,
    seed.included_toppings,
    seed.included_fruits,
    seed.included_extras,
    seed.tag,
    seed.image_url,
    seed.image_alt,
    seed.sort_order
FROM (
    VALUES
        ('combo-300', 'Combo 300 ml', 'size-300', 1290, 3, 0, 1, 'Seu primeiro Jade', 'https://images.unsplash.com/photo-1590301157890-4810ed352733?auto=format&fit=crop&w=1200&q=85', 'Açaí servido com acompanhamentos', 1),
        ('combo-500', 'Combo 500 ml', 'size-500', 1690, 3, 1, 1, 'Mais pedido', 'https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?auto=format&fit=crop&w=1200&q=85', 'Frutas frescas para acompanhar açaí', 2),
        ('combo-770', 'Combo 770 ml', 'size-770', 1990, 5, 1, 1, 'Pra matar a vontade', 'https://images.unsplash.com/photo-1511690743698-d9d85f2fbf38?auto=format&fit=crop&w=1200&q=85', 'Bowl generoso de açaí com frutas e acompanhamentos', 3),
        ('combo-marmita', 'Combo Marmita', 'size-1000', 2890, 6, 2, 1, 'Pra compartilhar', 'https://images.unsplash.com/photo-1505252585461-04db1eb84625?auto=format&fit=crop&w=1200&q=85', 'Açaí grande com frutas para compartilhar', 4)
) AS seed(combo_key, name, size_key, price_cents, included_toppings, included_fruits, included_extras, tag, image_url, image_alt, sort_order)
JOIN catalog_items AS sizes ON sizes.item_key = seed.size_key
ON CONFLICT (combo_key) DO NOTHING;

INSERT INTO catalog_rules (rule_key, rule_value)
VALUES
    ('included_toppings', '6'::jsonb),
    ('included_fruits', '1'::jsonb),
    ('delivery_fee_cents', '300'::jsonb),
    ('additional_topping_price_cents', '100'::jsonb),
    ('additional_fruit_price_cents', '200'::jsonb)
ON CONFLICT (rule_key) DO NOTHING;

ALTER TABLE catalog_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_combos ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_combo_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_rules ENABLE ROW LEVEL SECURITY;