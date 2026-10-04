package application

import (
	"context"
	"errors"
	"testing"
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
		EstimatedTotalCents: 1000,
		Items:               []OrderLineRequest{{ID: "line-1", Name: " "}},
	}
	if _, err := service.Create(context.Background(), request); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid order item error, got %v", err)
	}
}
