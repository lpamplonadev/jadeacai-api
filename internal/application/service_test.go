package application

import (
	"context"
	"errors"
	"testing"

	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

func TestValidateComboItemsRejectsMoreThan100Units(t *testing.T) {
	err := validateComboItems([]CatalogComboItemInput{
		{ItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7201", Quantity: 100},
		{ItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7202", Quantity: 1},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestValidateCatalogGourmetRequiresDescriptionAndRecipe(t *testing.T) {
	request := CreateCatalogComboRequest{
		Name:       "Banoffe",
		Category:   "gourmet",
		SizeItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7201",
		Items: []CatalogComboItemInput{
			{ItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7201", Quantity: 1},
			{ItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7202", Quantity: 1},
		},
		GourmetSizes: []CatalogComboGourmetSizeInput{{
			SizeItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7203", PriceCents: 2500,
		}},
	}
	if err := validateCatalogCombo(request); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected Gourmet without description to be rejected, got %v", err)
	}

	request.Description = "Açaí cremoso com banana e doce de leite."
	request.Items = request.Items[:1]
	if err := validateCatalogCombo(request); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected Gourmet without a fixed recipe to be rejected, got %v", err)
	}

	request.Items = append(request.Items, CatalogComboItemInput{
		ItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7202", Quantity: 1,
	})
	if err := validateCatalogCombo(request); err != nil {
		t.Fatalf("expected complete Gourmet product to be accepted, got %v", err)
	}
}

func TestValidateCatalogGourmetSizes(t *testing.T) {
	valid := []CatalogComboGourmetSizeInput{
		{SizeItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7201", PriceCents: 2500},
		{SizeItemID: "a4f535aa-8c2b-4f0f-9c31-783061cc7202", PriceCents: 3500},
	}
	if err := validateCatalogGourmetSizes(valid); err != nil {
		t.Fatalf("expected valid Gourmet sizes, got %v", err)
	}
	duplicate := append(valid, valid[0])
	if err := validateCatalogGourmetSizes(duplicate); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected duplicate Gourmet sizes to be rejected, got %v", err)
	}
	if err := validateCatalogGourmetSizes([]CatalogComboGourmetSizeInput{valid[0], {SizeItemID: valid[1].SizeItemID, PriceCents: -1}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected negative Gourmet prices to be rejected, got %v", err)
	}
}

func TestDashboardRejectsInvalidDateBeforeRepositoryCall(t *testing.T) {
	service := NewService(nil)
	if _, err := service.Dashboard(context.Background(), "2026-02-30"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}

func TestCreateOrderRejectsBlankLineNamesBeforeRepositoryCall(t *testing.T) {
	service := NewService(nil)
	request := CreateOrderRequest{
		Customer:            OrderCustomer{Name: "Ana", Phone: "21999990000"},
		Acai:                OrderAcai{FlavorID: "banana", SizeID: "500"},
		Delivery:            OrderDelivery{PostalCode: "20000-000", Street: "Rua Jade", Number: "10", Neighborhood: "Centro"},
		Payment:             OrderPayment{Method: "pix"},
		EstimatedTotalCents: 1000,
		Items:               []OrderLineRequest{{ID: "line-1", Name: " "}},
	}
	if _, err := service.Create(context.Background(), request); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid order item error, got %v", err)
	}
}

func TestCreateOrderRejectsInvalidPhoneBeforeRepositoryCall(t *testing.T) {
	service := NewService(nil)
	request := CreateOrderRequest{Customer: OrderCustomer{Phone: "2199990000"}}
	if _, err := service.Create(context.Background(), request); !errors.Is(err, domain.ErrInvalidPhone) {
		t.Fatalf("expected invalid phone error, got %v", err)
	}
}
