UPDATE catalog_rules
SET
    rule_value = jsonb_set(
        COALESCE(rule_value, '{}'::jsonb),
        '{deliveryZones}',
        '[]'::jsonb,
        true
    ),
    updated_at = now()
WHERE rule_key = 'store_settings';