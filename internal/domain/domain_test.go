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

func TestIsCatalogCategory(t *testing.T) {
	for _, category := range []string{"combo", "gourmet"} {
		if !IsCatalogCategory(category) {
			t.Errorf("expected %q to be a valid catalog category", category)
		}
	}
	if IsCatalogCategory("unknown") {
		t.Fatal("expected unknown catalog category to be rejected")
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

func TestIsValidBrazilianMobilePhone(t *testing.T) {
	for _, phone := range []string{"21999990000", "(21) 99999-0000"} {
		if !IsValidBrazilianMobilePhone(phone) {
			t.Errorf("expected %q to be a valid Brazilian mobile phone", phone)
		}
	}
	for _, phone := range []string{"(20) 99999-0000", "(21) 29999-0000", "(21) 9999-0000"} {
		if IsValidBrazilianMobilePhone(phone) {
			t.Errorf("expected %q to be rejected", phone)
		}
	}
}

func TestNormalizeBrazilianPhone(t *testing.T) {
	if got := NormalizeBrazilianPhone("(21) 99999-0000"); got != "21999990000" {
		t.Fatalf("expected normalized phone digits, got %q", got)
	}
}
