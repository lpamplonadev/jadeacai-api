INSERT INTO catalog_rules (rule_key, rule_value)
VALUES (
    'store_settings',
    '{
        "weeklyHours": {
            "monday": {"enabled": false, "opensAt": "", "closesAt": ""},
            "tuesday": {"enabled": true, "opensAt": "19:00", "closesAt": "23:00"},
            "wednesday": {"enabled": true, "opensAt": "19:00", "closesAt": "23:00"},
            "thursday": {"enabled": true, "opensAt": "19:00", "closesAt": "23:00"},
            "friday": {"enabled": true, "opensAt": "19:00", "closesAt": "23:00"},
            "saturday": {"enabled": true, "opensAt": "17:00", "closesAt": "23:00"},
            "sunday": {"enabled": true, "opensAt": "17:00", "closesAt": "23:00"}
        },
        "story": {
            "title": "Um intervalo gostoso muda o dia.",
            "body": "Açaí de verdade, feito com carinho em cada pedido."
        },
        "whatsAppNumber": "5521990174473",
        "manualOverride": null
    }'::jsonb
)
ON CONFLICT (rule_key) DO NOTHING;