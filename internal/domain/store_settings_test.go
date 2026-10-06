package domain

import (
	"testing"
	"time"
)

func TestDefaultStoreScheduleUsesSaoPauloHours(t *testing.T) {
	settings := DefaultStoreSettings()
	location, err := time.LoadLocation(StoreTimeZone)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		date time.Time
		open bool
	}{
		{name: "tuesday opens at 19", date: time.Date(2026, time.October, 6, 19, 0, 0, 0, location), open: true},
		{name: "friday before closing", date: time.Date(2026, time.October, 9, 22, 59, 0, 0, location), open: true},
		{name: "saturday opens at 17", date: time.Date(2026, time.October, 10, 17, 0, 0, 0, location), open: true},
		{name: "monday is closed", date: time.Date(2026, time.October, 5, 20, 0, 0, 0, location), open: false},
		{name: "closing time is exclusive", date: time.Date(2026, time.October, 6, 23, 0, 0, 0, location), open: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := StoreIsOpenAt(settings, test.date); got != test.open {
				t.Fatalf("expected open=%t, got %t", test.open, got)
			}
		})
	}
}

func TestManualStoreOverrideTakesPrecedence(t *testing.T) {
	settings := DefaultStoreSettings()
	closedMonday := time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)
	open := true
	settings.ManualOverride = &open
	if !StoreIsOpenAt(settings, closedMonday) {
		t.Fatal("manual open override should open the store")
	}

	closed := false
	settings.ManualOverride = &closed
	if StoreIsOpenAt(settings, time.Date(2026, time.October, 6, 20, 0, 0, 0, time.UTC)) {
		t.Fatal("manual closed override should close the store")
	}
}

func TestValidateStoreSettingsRejectsInvalidHoursAndWhatsAppNumber(t *testing.T) {
	settings := DefaultStoreSettings()
	settings.WeeklyHours["tuesday"] = OperatingHours{Enabled: true, OpensAt: "23:00", ClosesAt: "19:00"}
	if ValidateStoreSettings(settings) == nil {
		t.Fatal("expected invalid opening interval to be rejected")
	}

	settings = DefaultStoreSettings()
	settings.WhatsAppNumber = "552199999"
	if ValidateStoreSettings(settings) == nil {
		t.Fatal("expected invalid WhatsApp number to be rejected")
	}
}
