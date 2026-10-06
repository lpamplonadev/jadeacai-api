package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
)

func (store *Store) CreateCatalogGourmet(ctx context.Context, request createCatalogGourmetRequest, gourmetKey string, available bool) (catalogGourmetRecord, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("begin create catalog Gourmet: %w", err)
	}
	defer tx.Rollback()

	if err := validateCatalogGourmetReferences(ctx, tx, request.Items, nil, request.Sizes, nil); err != nil {
		return catalogGourmetRecord{}, err
	}
	var gourmetID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO catalog_gourmet (
			gourmet_key, name, description, tag, image_url, image_alt, available, sort_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, gourmetKey, strings.TrimSpace(request.Name), strings.TrimSpace(request.Description), request.Tag, request.ImageURL, request.ImageAlt, available, request.SortOrder).Scan(&gourmetID)
	if err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("insert catalog Gourmet: %w", err)
	}
	if err := insertCatalogGourmetItems(ctx, tx, gourmetID, request.Items); err != nil {
		return catalogGourmetRecord{}, err
	}
	if err := insertCatalogGourmetSizes(ctx, tx, gourmetID, request.Sizes); err != nil {
		return catalogGourmetRecord{}, err
	}
	gourmet, err := loadCatalogGourmet(ctx, tx, gourmetID)
	if err != nil {
		return catalogGourmetRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("commit catalog Gourmet: %w", err)
	}
	return gourmet, nil
}

func (store *Store) UpdateCatalogGourmet(ctx context.Context, gourmetID string, request updateCatalogGourmetRequest) (catalogGourmetRecord, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalogGourmetRecord{}, false, fmt.Errorf("begin update catalog Gourmet: %w", err)
	}
	defer tx.Rollback()

	gourmet, err := loadCatalogGourmet(ctx, tx, gourmetID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && gourmet.DeletedAt != nil) {
		return catalogGourmetRecord{}, false, nil
	}
	if err != nil {
		return catalogGourmetRecord{}, false, err
	}
	items := make([]application.CatalogComboItemInput, 0, len(gourmet.Items))
	for _, item := range gourmet.Items {
		items = append(items, application.CatalogComboItemInput{ItemID: item.ItemID, Quantity: item.Quantity})
	}
	if request.Items != nil {
		items = *request.Items
	}
	sizes := make([]application.CatalogGourmetSizeInput, 0, len(gourmet.Sizes))
	for _, size := range gourmet.Sizes {
		sizes = append(sizes, application.CatalogGourmetSizeInput{SizeItemID: size.SizeItemID, PriceCents: size.PriceCents})
	}
	if request.Sizes != nil {
		sizes = *request.Sizes
	}
	if len(items) < 2 || len(sizes) == 0 {
		return catalogGourmetRecord{}, false, application.ErrInvalidInput
	}
	if err := validateCatalogGourmetReferences(ctx, tx, items, gourmet.Items, sizes, gourmet.Sizes); err != nil {
		return catalogGourmetRecord{}, false, err
	}
	if request.Name != nil {
		gourmet.Name = strings.TrimSpace(*request.Name)
	}
	if request.Description != nil {
		gourmet.Description = strings.TrimSpace(*request.Description)
	}
	if request.Tag != nil {
		gourmet.Tag = *request.Tag
	}
	if request.ImageURL != nil {
		gourmet.ImageURL = *request.ImageURL
	}
	if request.ImageAlt != nil {
		gourmet.ImageAlt = *request.ImageAlt
	}
	if request.Available != nil {
		gourmet.Available = *request.Available
	}
	if request.SortOrder != nil {
		gourmet.SortOrder = *request.SortOrder
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE catalog_gourmet
		SET name = $1, description = $2, tag = $3, image_url = $4, image_alt = $5,
			available = $6, sort_order = $7, updated_at = now()
		WHERE id = $8::uuid AND deleted_at IS NULL
	`, gourmet.Name, gourmet.Description, gourmet.Tag, gourmet.ImageURL, gourmet.ImageAlt, gourmet.Available, gourmet.SortOrder, gourmetID); err != nil {
		return catalogGourmetRecord{}, false, fmt.Errorf("update catalog Gourmet: %w", err)
	}
	if request.Items != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_gourmet_items WHERE gourmet_id = $1::uuid`, gourmetID); err != nil {
			return catalogGourmetRecord{}, false, fmt.Errorf("replace Gourmet recipe: %w", err)
		}
		if err := insertCatalogGourmetItems(ctx, tx, gourmetID, items); err != nil {
			return catalogGourmetRecord{}, false, err
		}
	}
	if request.Sizes != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_gourmet_sizes WHERE gourmet_id = $1::uuid`, gourmetID); err != nil {
			return catalogGourmetRecord{}, false, fmt.Errorf("replace Gourmet sizes: %w", err)
		}
		if err := insertCatalogGourmetSizes(ctx, tx, gourmetID, sizes); err != nil {
			return catalogGourmetRecord{}, false, err
		}
	}
	updated, err := loadCatalogGourmet(ctx, tx, gourmetID)
	if err != nil {
		return catalogGourmetRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return catalogGourmetRecord{}, false, fmt.Errorf("commit catalog Gourmet update: %w", err)
	}
	return updated, true, nil
}

func (store *Store) ArchiveCatalogGourmet(ctx context.Context, gourmetID string) (bool, error) {
	var archivedID string
	err := store.db.QueryRowContext(ctx, `
		UPDATE catalog_gourmet
		SET available = false, deleted_at = now(), updated_at = now()
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING id::text
	`, gourmetID).Scan(&archivedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("archive catalog Gourmet: %w", err)
	}
	return true, nil
}

func loadCatalogGourmet(ctx context.Context, queryer catalogQueryer, gourmetID string) (catalogGourmetRecord, error) {
	var gourmet catalogGourmetRecord
	var deletedAt sql.NullTime
	err := queryer.QueryRowContext(ctx, `
		SELECT id::text, gourmet_key, name, description, tag, image_url, image_alt,
			available, sort_order, deleted_at
		FROM catalog_gourmet
		WHERE id = $1::uuid
	`, gourmetID).Scan(&gourmet.ID, &gourmet.GourmetKey, &gourmet.Name, &gourmet.Description, &gourmet.Tag, &gourmet.ImageURL, &gourmet.ImageAlt, &gourmet.Available, &gourmet.SortOrder, &deletedAt)
	if err != nil {
		return catalogGourmetRecord{}, err
	}
	gourmet.DeletedAt = formatCatalogTimestamp(deletedAt)
	gourmet.Items = make([]catalogComboItemRecord, 0)
	itemRows, err := queryer.QueryContext(ctx, `
		SELECT item.id::text, item.item_key, item.kind, item.name, relation.quantity,
			item.available AND item.deleted_at IS NULL
		FROM catalog_gourmet_items AS relation
		JOIN catalog_items AS item ON item.id = relation.item_id
		WHERE relation.gourmet_id = $1::uuid
		ORDER BY relation.sort_order, item.sort_order, item.name
	`, gourmetID)
	if err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("query Gourmet recipe: %w", err)
	}
	for itemRows.Next() {
		var item catalogComboItemRecord
		if err := itemRows.Scan(&item.ItemID, &item.ItemKey, &item.Kind, &item.Name, &item.Quantity, &item.Available); err != nil {
			itemRows.Close()
			return catalogGourmetRecord{}, fmt.Errorf("scan Gourmet recipe: %w", err)
		}
		gourmet.Items = append(gourmet.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		itemRows.Close()
		return catalogGourmetRecord{}, fmt.Errorf("iterate Gourmet recipe: %w", err)
	}
	if err := itemRows.Close(); err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("close Gourmet recipe: %w", err)
	}
	gourmet.Sizes = make([]catalogGourmetSizeRecord, 0)
	sizeRows, err := queryer.QueryContext(ctx, `
		SELECT item.id::text, item.name, sizes.price_cents,
			item.available AND item.deleted_at IS NULL
		FROM catalog_gourmet_sizes AS sizes
		JOIN catalog_items AS item ON item.id = sizes.size_item_id
		WHERE sizes.gourmet_id = $1::uuid
		ORDER BY sizes.sort_order, item.sort_order, item.name
	`, gourmetID)
	if err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("query Gourmet sizes: %w", err)
	}
	defer sizeRows.Close()
	for sizeRows.Next() {
		var size catalogGourmetSizeRecord
		if err := sizeRows.Scan(&size.SizeItemID, &size.SizeName, &size.PriceCents, &size.Available); err != nil {
			return catalogGourmetRecord{}, fmt.Errorf("scan Gourmet size: %w", err)
		}
		gourmet.Sizes = append(gourmet.Sizes, size)
	}
	if err := sizeRows.Err(); err != nil {
		return catalogGourmetRecord{}, fmt.Errorf("iterate Gourmet sizes: %w", err)
	}
	return gourmet, nil
}

func insertCatalogGourmetItems(ctx context.Context, tx *sql.Tx, gourmetID string, items []application.CatalogComboItemInput) error {
	for index, item := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_gourmet_items (gourmet_id, item_id, quantity, sort_order)
			VALUES ($1::uuid, $2::uuid, $3, $4)
		`, gourmetID, item.ItemID, item.Quantity, index); err != nil {
			return fmt.Errorf("insert Gourmet recipe item: %w", err)
		}
	}
	return nil
}

func insertCatalogGourmetSizes(ctx context.Context, tx *sql.Tx, gourmetID string, sizes []application.CatalogGourmetSizeInput) error {
	for index, size := range sizes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_gourmet_sizes (gourmet_id, size_item_id, price_cents, sort_order)
			VALUES ($1::uuid, $2::uuid, $3, $4)
		`, gourmetID, size.SizeItemID, size.PriceCents, index); err != nil {
			return fmt.Errorf("insert Gourmet size: %w", err)
		}
	}
	return nil
}

func validateCatalogGourmetReferences(ctx context.Context, queryer catalogQueryer, items []application.CatalogComboItemInput, previousItems []catalogComboItemRecord, sizes []application.CatalogGourmetSizeInput, previousSizes []catalogGourmetSizeRecord) error {
	previousItemIDs := make(map[string]struct{}, len(previousItems))
	for _, item := range previousItems {
		previousItemIDs[item.ItemID] = struct{}{}
	}
	flavorCount := 0
	for _, item := range items {
		var kind string
		var available bool
		if err := queryer.QueryRowContext(ctx, `
			SELECT kind, available AND deleted_at IS NULL
			FROM catalog_items WHERE id = $1::uuid
		`, item.ItemID).Scan(&kind, &available); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return application.ErrCatalogReference
			}
			return fmt.Errorf("validate Gourmet recipe item: %w", err)
		}
		if kind == "size" {
			return application.ErrCatalogReference
		}
		if kind == "flavor" {
			flavorCount++
		}
		if !available {
			if _, existed := previousItemIDs[item.ItemID]; !existed {
				return application.ErrCatalogReference
			}
		}
	}
	if flavorCount != 1 {
		return application.ErrInvalidInput
	}
	previousSizeIDs := make(map[string]struct{}, len(previousSizes))
	for _, size := range previousSizes {
		previousSizeIDs[size.SizeItemID] = struct{}{}
	}
	for _, size := range sizes {
		var isActiveSize bool
		if err := queryer.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM catalog_items
				WHERE id = $1::uuid AND kind = 'size' AND available AND deleted_at IS NULL
			)
		`, size.SizeItemID).Scan(&isActiveSize); err != nil {
			return fmt.Errorf("validate Gourmet size: %w", err)
		}
		if !isActiveSize {
			if _, existed := previousSizeIDs[size.SizeItemID]; !existed {
				return application.ErrCatalogReference
			}
		}
	}
	return nil
}
