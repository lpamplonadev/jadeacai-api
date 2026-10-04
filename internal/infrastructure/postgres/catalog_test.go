package postgres

import (
	"context"
	"testing"

	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
)

func TestCatalogComboCanRetainPreviouslyLinkedInactiveItem(t *testing.T) {
	itemID := "a4f535aa-8c2b-4f0f-9c31-783061cc7201"
	err := validateCatalogReferences(
		context.Background(),
		nil,
		"a4f535aa-8c2b-4f0f-9c31-783061cc7202",
		[]application.CatalogComboItemInput{{ItemID: itemID, Quantity: 1}},
		"a4f535aa-8c2b-4f0f-9c31-783061cc7202",
		[]application.CatalogComboItemRecord{{ItemID: itemID, Available: false}},
	)
	if err != nil {
		t.Fatalf("expected an existing inactive link to remain valid, got %v", err)
	}
}
