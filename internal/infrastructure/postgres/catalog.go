package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

type catalogQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type catalogRowScanner interface {
	Scan(...any) error
}

func (store *Store) Catalog(ctx context.Context) (catalogData, error) {
	result := catalogData{
		Items:    make([]catalogItemRecord, 0),
		Combos:   make([]catalogComboRecord, 0),
		Gourmets: make([]catalogGourmetRecord, 0),
		Rules:    make([]catalogRuleRecord, 0),
	}

	items, err := store.db.QueryContext(ctx, `
		SELECT id::text, item_key, kind, name, description, image_url, image_alt,
			price_cents, available, sort_order, deleted_at
		FROM catalog_items
		ORDER BY kind, sort_order, name
	`)
	if err != nil {
		return catalogData{}, fmt.Errorf("query catalog items: %w", err)
	}
	for items.Next() {
		item, err := scanCatalogItem(items)
		if err != nil {
			items.Close()
			return catalogData{}, fmt.Errorf("scan catalog item: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := items.Err(); err != nil {
		items.Close()
		return catalogData{}, fmt.Errorf("iterate catalog items: %w", err)
	}
	items.Close()

	combos, err := store.db.QueryContext(ctx, `
		SELECT c.id::text, c.combo_key, c.name, c.size_item_id::text, size.name,
			c.price_cents, c.included_toppings, c.included_fruits, c.included_extras,
			c.tag, c.image_url, c.image_alt, c.available, c.sort_order, c.deleted_at
		FROM catalog_combos AS c
	JOIN catalog_items AS size ON size.id = c.size_item_id
	ORDER BY c.sort_order, c.name
	`)
	if err != nil {
		return catalogData{}, fmt.Errorf("query catalog combos: %w", err)
	}
	comboIndexes := make(map[string]int)
	for combos.Next() {
		var combo catalogComboRecord
		var deletedAt sql.NullTime
		if err := combos.Scan(
			&combo.ID,
			&combo.ComboKey,
			&combo.Name,
			&combo.SizeItemID,
			&combo.SizeName,
			&combo.PriceCents,
			&combo.IncludedToppings,
			&combo.IncludedFruits,
			&combo.IncludedExtras,
			&combo.Tag,
			&combo.ImageURL,
			&combo.ImageAlt,
			&combo.Available,
			&combo.SortOrder,
			&deletedAt,
		); err != nil {
			combos.Close()
			return catalogData{}, fmt.Errorf("scan catalog combo: %w", err)
		}
		combo.DeletedAt = formatCatalogTimestamp(deletedAt)
		combo.Items = make([]catalogComboItemRecord, 0)
		comboIndexes[combo.ID] = len(result.Combos)
		result.Combos = append(result.Combos, combo)
	}
	if err := combos.Err(); err != nil {
		combos.Close()
		return catalogData{}, fmt.Errorf("iterate catalog combos: %w", err)
	}
	combos.Close()

	comboItems, err := store.db.QueryContext(ctx, `
		SELECT relation.combo_id::text, item.id::text, item.item_key, item.kind,
			item.name, relation.quantity, item.available AND item.deleted_at IS NULL
		FROM catalog_combo_items AS relation
	JOIN catalog_items AS item ON item.id = relation.item_id
	ORDER BY item.kind, item.sort_order, item.name
	`)
	if err != nil {
		return catalogData{}, fmt.Errorf("query catalog combo items: %w", err)
	}
	for comboItems.Next() {
		var comboID string
		var item catalogComboItemRecord
		if err := comboItems.Scan(&comboID, &item.ItemID, &item.ItemKey, &item.Kind, &item.Name, &item.Quantity, &item.Available); err != nil {
			comboItems.Close()
			return catalogData{}, fmt.Errorf("scan catalog combo item: %w", err)
		}
		if index, exists := comboIndexes[comboID]; exists {
			result.Combos[index].Items = append(result.Combos[index].Items, item)
		}
	}
	if err := comboItems.Err(); err != nil {
		comboItems.Close()
		return catalogData{}, fmt.Errorf("iterate catalog combo items: %w", err)
	}
	comboItems.Close()

	rules, err := store.db.QueryContext(ctx, `SELECT rule_key, rule_value FROM catalog_rules ORDER BY rule_key`)
	if err != nil {
		return catalogData{}, fmt.Errorf("query catalog rules: %w", err)
	}
	defer rules.Close()
	for rules.Next() {
		var rule catalogRuleRecord
		var rawValue []byte
		if err := rules.Scan(&rule.Key, &rawValue); err != nil {
			return catalogData{}, fmt.Errorf("scan catalog rule: %w", err)
		}
		if rule.Key == domain.StoreSettingsRuleKey {
			continue
		}
		if err := json.Unmarshal(rawValue, &rule.Value); err != nil {
			return catalogData{}, fmt.Errorf("decode catalog rule: %w", err)
		}
		result.Rules = append(result.Rules, rule)
	}
	if err := rules.Err(); err != nil {
		return catalogData{}, fmt.Errorf("iterate catalog rules: %w", err)
	}

	gourmetRows, err := store.db.QueryContext(ctx, `
		SELECT id::text
		FROM catalog_gourmet
		ORDER BY sort_order, name
	`)
	if err != nil {
		return catalogData{}, fmt.Errorf("query catalog Gourmets: %w", err)
	}
	gourmetIDs := make([]string, 0)
	for gourmetRows.Next() {
		var gourmetID string
		if err := gourmetRows.Scan(&gourmetID); err != nil {
			gourmetRows.Close()
			return catalogData{}, fmt.Errorf("scan catalog Gourmet id: %w", err)
		}
		gourmetIDs = append(gourmetIDs, gourmetID)
	}
	if err := gourmetRows.Err(); err != nil {
		gourmetRows.Close()
		return catalogData{}, fmt.Errorf("iterate catalog Gourmet ids: %w", err)
	}
	gourmetRows.Close()
	for _, gourmetID := range gourmetIDs {
		gourmet, err := loadCatalogGourmet(ctx, store.db, gourmetID)
		if err != nil {
			return catalogData{}, fmt.Errorf("load catalog Gourmet: %w", err)
		}
		result.Gourmets = append(result.Gourmets, gourmet)
	}

	return result, nil
}

func (store *Store) CreateCatalogItem(ctx context.Context, request createCatalogItemRequest, itemKey string, available bool) (catalogItemRecord, error) {
	const query = `
		INSERT INTO catalog_items (item_key, kind, name, description, image_url, image_alt, price_cents, available, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, item_key, kind, name, description, image_url, image_alt, price_cents, available, sort_order, deleted_at
	`
	item, err := scanCatalogItem(store.db.QueryRowContext(ctx, query,
		itemKey, request.Kind, request.Name, request.Description, request.ImageURL, request.ImageAlt,
		request.PriceCents, available, request.SortOrder,
	))
	if err != nil {
		return catalogItemRecord{}, fmt.Errorf("insert catalog item: %w", err)
	}
	return item, nil
}

func (store *Store) UpdateCatalogItem(ctx context.Context, itemID string, request updateCatalogItemRequest) (catalogItemRecord, bool, error) {
	const query = `
		UPDATE catalog_items
		SET name = COALESCE($1, name),
			description = COALESCE($2, description),
			image_url = COALESCE($3, image_url),
			image_alt = COALESCE($4, image_alt),
			price_cents = COALESCE($5, price_cents),
			available = COALESCE($6, available),
			sort_order = COALESCE($7, sort_order),
			updated_at = now()
		WHERE id = $8::uuid AND deleted_at IS NULL
		RETURNING id::text, item_key, kind, name, description, image_url, image_alt, price_cents, available, sort_order, deleted_at
	`
	item, err := scanCatalogItem(store.db.QueryRowContext(ctx, query,
		catalogPointerValue(request.Name),
		catalogPointerValue(request.Description),
		catalogPointerValue(request.ImageURL),
		catalogPointerValue(request.ImageAlt),
		catalogPointerValue(request.PriceCents),
		catalogPointerValue(request.Available),
		catalogPointerValue(request.SortOrder),
		itemID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return catalogItemRecord{}, false, nil
	}
	if err != nil {
		return catalogItemRecord{}, false, fmt.Errorf("update catalog item: %w", err)
	}
	return item, true, nil
}

func (store *Store) ArchiveCatalogItem(ctx context.Context, itemID string) (bool, error) {
	var archivedID string
	err := store.db.QueryRowContext(ctx, `
		UPDATE catalog_items
		SET available = false, deleted_at = now(), updated_at = now()
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING id::text
	`, itemID).Scan(&archivedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("archive catalog item: %w", err)
	}
	return true, nil
}

func (store *Store) CreateCatalogCombo(ctx context.Context, request createCatalogComboRequest, comboKey string, available bool) (catalogComboRecord, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalogComboRecord{}, fmt.Errorf("begin create catalog combo: %w", err)
	}
	defer tx.Rollback()

	if err := validateCatalogReferences(ctx, tx, request.SizeItemID, request.Items, "", nil); err != nil {
		return catalogComboRecord{}, err
	}
	const query = `
		INSERT INTO catalog_combos (
			combo_key, name, size_item_id, price_cents, included_toppings,
			included_fruits, included_extras, tag, image_url, image_alt, available, sort_order
		)
		VALUES ($1, $2, $3, $4::uuid, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text
	`
	var comboID string
	err = tx.QueryRowContext(ctx, query,
		comboKey,
		strings.TrimSpace(request.Name),
		request.SizeItemID,
		request.PriceCents,
		request.IncludedToppings,
		request.IncludedFruits,
		request.IncludedExtras,
		request.Tag,
		request.ImageURL,
		request.ImageAlt,
		available,
		request.SortOrder,
	).Scan(&comboID)
	if err != nil {
		return catalogComboRecord{}, fmt.Errorf("insert catalog combo: %w", err)
	}
	if err := insertCatalogComboItems(ctx, tx, comboID, request.Items); err != nil {
		return catalogComboRecord{}, err
	}
	combo, err := loadCatalogCombo(ctx, tx, comboID)
	if err != nil {
		return catalogComboRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalogComboRecord{}, fmt.Errorf("commit catalog combo: %w", err)
	}
	return combo, nil
}

func (store *Store) UpdateCatalogCombo(ctx context.Context, comboID string, request updateCatalogComboRequest) (catalogComboRecord, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalogComboRecord{}, false, fmt.Errorf("begin update catalog combo: %w", err)
	}
	defer tx.Rollback()

	combo, err := loadCatalogCombo(ctx, tx, comboID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && combo.DeletedAt != nil) {
		return catalogComboRecord{}, false, nil
	}
	if err != nil {
		return catalogComboRecord{}, false, err
	}

	previousSizeItemID := combo.SizeItemID
	previousItems := combo.Items
	if request.SizeItemID != nil {
		combo.SizeItemID = *request.SizeItemID
	}
	items := make([]catalogComboItemInput, 0, len(combo.Items))
	for _, item := range combo.Items {
		items = append(items, catalogComboItemInput{ItemID: item.ItemID, Quantity: item.Quantity})
	}
	if request.Items != nil {
		items = *request.Items
	}
	if err := validateCatalogReferences(ctx, tx, combo.SizeItemID, items, previousSizeItemID, previousItems); err != nil {
		return catalogComboRecord{}, false, err
	}

	if request.Name != nil {
		combo.Name = strings.TrimSpace(*request.Name)
	}
	if request.PriceCents != nil {
		combo.PriceCents = *request.PriceCents
	}
	if request.IncludedToppings != nil {
		combo.IncludedToppings = *request.IncludedToppings
	}
	if request.IncludedFruits != nil {
		combo.IncludedFruits = *request.IncludedFruits
	}
	if request.IncludedExtras != nil {
		combo.IncludedExtras = *request.IncludedExtras
	}
	if request.Tag != nil {
		combo.Tag = *request.Tag
	}
	if request.ImageURL != nil {
		combo.ImageURL = *request.ImageURL
	}
	if request.ImageAlt != nil {
		combo.ImageAlt = *request.ImageAlt
	}
	if request.Available != nil {
		combo.Available = *request.Available
	}
	if request.SortOrder != nil {
		combo.SortOrder = *request.SortOrder
	}
	const updateQuery = `
		UPDATE catalog_combos
		SET name = $1, size_item_id = $2::uuid, price_cents = $3,
			included_toppings = $4, included_fruits = $5, included_extras = $6,
			tag = $7, image_url = $8, image_alt = $9, available = $10,
			sort_order = $11, updated_at = now()
		WHERE id = $12::uuid AND deleted_at IS NULL
	`
	if _, err := tx.ExecContext(ctx, updateQuery,
		combo.Name,
		combo.SizeItemID,
		combo.PriceCents,
		combo.IncludedToppings,
		combo.IncludedFruits,
		combo.IncludedExtras,
		combo.Tag,
		combo.ImageURL,
		combo.ImageAlt,
		combo.Available,
		combo.SortOrder,
		comboID,
	); err != nil {
		return catalogComboRecord{}, false, fmt.Errorf("update catalog combo: %w", err)
	}
	if request.Items != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_combo_items WHERE combo_id = $1::uuid`, comboID); err != nil {
			return catalogComboRecord{}, false, fmt.Errorf("replace catalog combo items: %w", err)
		}
		if err := insertCatalogComboItems(ctx, tx, comboID, *request.Items); err != nil {
			return catalogComboRecord{}, false, err
		}
	}
	updated, err := loadCatalogCombo(ctx, tx, comboID)
	if err != nil {
		return catalogComboRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return catalogComboRecord{}, false, fmt.Errorf("commit catalog combo: %w", err)
	}
	return updated, true, nil
}

func (store *Store) ArchiveCatalogCombo(ctx context.Context, comboID string) (bool, error) {
	var archivedID string
	err := store.db.QueryRowContext(ctx, `
		UPDATE catalog_combos
		SET available = false, deleted_at = now(), updated_at = now()
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING id::text
	`, comboID).Scan(&archivedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("archive catalog combo: %w", err)
	}
	return true, nil
}

func scanCatalogItem(scanner catalogRowScanner) (catalogItemRecord, error) {
	var item catalogItemRecord
	var deletedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.ItemKey,
		&item.Kind,
		&item.Name,
		&item.Description,
		&item.ImageURL,
		&item.ImageAlt,
		&item.PriceCents,
		&item.Available,
		&item.SortOrder,
		&deletedAt,
	); err != nil {
		return catalogItemRecord{}, err
	}
	item.DeletedAt = formatCatalogTimestamp(deletedAt)
	return item, nil
}

func loadCatalogCombo(ctx context.Context, queryer catalogQueryer, comboID string) (catalogComboRecord, error) {
	var combo catalogComboRecord
	var deletedAt sql.NullTime
	err := queryer.QueryRowContext(ctx, `
		SELECT c.id::text, c.combo_key, c.name, c.size_item_id::text, size.name,
		c.price_cents, c.included_toppings, c.included_fruits, c.included_extras,
			c.tag, c.image_url, c.image_alt, c.available, c.sort_order, c.deleted_at
		FROM catalog_combos AS c
		JOIN catalog_items AS size ON size.id = c.size_item_id
		WHERE c.id = $1::uuid
	`, comboID).Scan(
		&combo.ID,
		&combo.ComboKey,
		&combo.Name,
		&combo.SizeItemID,
		&combo.SizeName,
		&combo.PriceCents,
		&combo.IncludedToppings,
		&combo.IncludedFruits,
		&combo.IncludedExtras,
		&combo.Tag,
		&combo.ImageURL,
		&combo.ImageAlt,
		&combo.Available,
		&combo.SortOrder,
		&deletedAt,
	)
	if err != nil {
		return catalogComboRecord{}, err
	}
	combo.DeletedAt = formatCatalogTimestamp(deletedAt)
	combo.Items = make([]catalogComboItemRecord, 0)
	rows, err := queryer.QueryContext(ctx, `
		SELECT item.id::text, item.item_key, item.kind, item.name, relation.quantity,
			item.available AND item.deleted_at IS NULL
		FROM catalog_combo_items AS relation
		JOIN catalog_items AS item ON item.id = relation.item_id
		WHERE relation.combo_id = $1::uuid
		ORDER BY item.kind, item.sort_order, item.name
	`, comboID)
	if err != nil {
		return catalogComboRecord{}, fmt.Errorf("query combo item links: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item catalogComboItemRecord
		if err := rows.Scan(&item.ItemID, &item.ItemKey, &item.Kind, &item.Name, &item.Quantity, &item.Available); err != nil {
			return catalogComboRecord{}, fmt.Errorf("scan combo item link: %w", err)
		}
		combo.Items = append(combo.Items, item)
	}
	if err := rows.Err(); err != nil {
		return catalogComboRecord{}, fmt.Errorf("iterate combo item links: %w", err)
	}
	if err := rows.Close(); err != nil {
		return catalogComboRecord{}, fmt.Errorf("close combo item links: %w", err)
	}
	return combo, nil
}

func validateCatalogReferences(ctx context.Context, queryer catalogQueryer, sizeItemID string, items []catalogComboItemInput, previousSizeItemID string, previousItems []catalogComboItemRecord) error {
	if sizeItemID != previousSizeItemID {
		var isActiveSize bool
		if err := queryer.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM catalog_items
			WHERE id = $1::uuid AND kind = 'size' AND available AND deleted_at IS NULL
		)
	`, sizeItemID).Scan(&isActiveSize); err != nil {
			return fmt.Errorf("validate combo size: %w", err)
		}
		if !isActiveSize {
			return application.ErrCatalogReference
		}
	}

	previousItemIDs := make(map[string]struct{}, len(previousItems))
	for _, item := range previousItems {
		previousItemIDs[item.ItemID] = struct{}{}
	}
	for _, item := range items {
		if _, wasAlreadyLinked := previousItemIDs[item.ItemID]; wasAlreadyLinked {
			continue
		}
		var isSelectable bool
		if err := queryer.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM catalog_items
				WHERE id = $1::uuid AND available AND deleted_at IS NULL
			)
		`, item.ItemID).Scan(&isSelectable); err != nil {
			return fmt.Errorf("validate combo item: %w", err)
		}
		if !isSelectable {
			return application.ErrCatalogReference
		}
	}
	return nil
}

func insertCatalogComboItems(ctx context.Context, tx *sql.Tx, comboID string, items []catalogComboItemInput) error {
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_combo_items (combo_id, item_id, quantity)
			VALUES ($1::uuid, $2::uuid, $3)
		`, comboID, item.ItemID, item.Quantity); err != nil {
			return fmt.Errorf("insert combo item link: %w", err)
		}
	}
	return nil
}

func formatCatalogTimestamp(timestamp sql.NullTime) *string {
	if !timestamp.Valid {
		return nil
	}
	formatted := timestamp.Time.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func catalogPointerValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}
