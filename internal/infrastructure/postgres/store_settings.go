package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

func (store *Store) StoreSettings(ctx context.Context) (domain.StoreSettings, error) {
	var rawValue []byte
	err := store.db.QueryRowContext(ctx, `
		SELECT rule_value
		FROM catalog_rules
		WHERE rule_key = $1
	`, domain.StoreSettingsRuleKey).Scan(&rawValue)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DefaultStoreSettings(), nil
	}
	if err != nil {
		return domain.StoreSettings{}, fmt.Errorf("query store settings: %w", err)
	}

	var settings domain.StoreSettings
	if err := json.Unmarshal(rawValue, &settings); err != nil {
		return domain.StoreSettings{}, fmt.Errorf("decode store settings: %w", err)
	}
	return settings, nil
}

func (store *Store) SaveStoreSettings(ctx context.Context, settings domain.StoreSettings) error {
	rawValue, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode store settings: %w", err)
	}
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO catalog_rules (rule_key, rule_value, updated_at)
		VALUES ($1, $2::jsonb, now())
		ON CONFLICT (rule_key) DO UPDATE
		SET rule_value = EXCLUDED.rule_value, updated_at = now()
	`, domain.StoreSettingsRuleKey, rawValue)
	if err != nil {
		return fmt.Errorf("save store settings: %w", err)
	}
	return nil
}
