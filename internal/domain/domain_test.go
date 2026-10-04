package domain

import "testing"

func TestIsValidOrderStatus(t *testing.T) {
	if !IsValidOrderStatus(string(OrderStatusPreparing)) {
		t.Fatal("expected preparing to be a valid order status")
	}
	if IsValidOrderStatus("unknown") {
		t.Fatal("expected unknown status to be rejected")
	}
}

func TestIsCatalogItemKind(t *testing.T) {
	if !IsCatalogItemKind(string(CatalogTopping)) {
		t.Fatal("expected topping to be a valid catalog kind")
	}
	if IsCatalogItemKind("unknown") {
		t.Fatal("expected unknown catalog kind to be rejected")
	}
}

func TestIsValidUUID(t *testing.T) {
	if !IsValidUUID("a4f535aa-8c2b-4f0f-9c31-783061cc7201") {
		t.Fatal("expected valid UUID to be accepted")
	}
	if IsValidUUID("not-a-uuid") {
		t.Fatal("expected malformed UUID to be rejected")
	}
}
