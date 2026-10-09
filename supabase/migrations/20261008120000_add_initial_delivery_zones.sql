WITH default_zones (name, zone) AS (
    VALUES
        ('Padre Miguel', jsonb_build_object(
            'name', 'Padre Miguel',
            'neighborhoods', jsonb_build_array('Padre Miguel'),
            'feeCents', 0,
            'enabled', false
        )),
        ('Bangu', jsonb_build_object(
            'name', 'Bangu',
            'neighborhoods', jsonb_build_array('Bangu'),
            'feeCents', 0,
            'enabled', false
        )),
        ('Sulacap', jsonb_build_object(
            'name', 'Sulacap',
            'neighborhoods', jsonb_build_array('Sulacap'),
            'feeCents', 0,
            'enabled', false
        )),
        ('Magalh' || chr(227) || 'es', jsonb_build_object(
            'name', 'Magalh' || chr(227) || 'es',
            'neighborhoods', jsonb_build_array('Magalh' || chr(227) || 'es Bastos'),
            'feeCents', 0,
            'enabled', false
        ))
), current_settings AS (
    SELECT
        rule_key,
        rule_value,
        CASE
            WHEN jsonb_typeof(rule_value -> 'deliveryZones') = 'array'
                THEN rule_value -> 'deliveryZones'
            ELSE '[]'::jsonb
        END AS delivery_zones
    FROM catalog_rules
    WHERE rule_key = 'store_settings'
), merged_zones AS (
    SELECT
        current_settings.rule_key,
        COALESCE(jsonb_agg(zones.zone), '[]'::jsonb) AS delivery_zones
    FROM current_settings
    CROSS JOIN LATERAL (
        SELECT existing.zone
        FROM jsonb_array_elements(current_settings.delivery_zones) AS existing(zone)

        UNION ALL

        SELECT defaults.zone
        FROM default_zones AS defaults
        WHERE NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(current_settings.delivery_zones) AS existing(zone)
            WHERE lower(btrim(existing.zone ->> 'name')) = lower(defaults.name)
        )
    ) AS zones
    GROUP BY current_settings.rule_key
)
UPDATE catalog_rules AS settings
SET
    rule_value = jsonb_set(
        COALESCE(settings.rule_value, '{}'::jsonb),
        '{deliveryZones}',
        merged_zones.delivery_zones,
        true
    ),
    updated_at = now()
FROM merged_zones
WHERE settings.rule_key = merged_zones.rule_key;