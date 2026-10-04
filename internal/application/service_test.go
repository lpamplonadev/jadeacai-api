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
