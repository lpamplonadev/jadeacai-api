package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	StoreSettingsRuleKey = "store_settings"
	StoreTimeZone        = "America/Sao_Paulo"
	DeliveryOrigin       = "Rua Nepomuceno, 12, Realengo, Rio de Janeiro - RJ"
)

var ErrInvalidStoreSettings = errors.New("invalid store settings")
var ErrStoreClosed = errors.New("store is closed")
var ErrDeliveryAreaUnavailable = errors.New("delivery area is unavailable")

type OperatingHours struct {
	Enabled  bool   `json:"enabled"`
	OpensAt  string `json:"opensAt"`
	ClosesAt string `json:"closesAt"`
}

type StoreStory struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type DeliveryZone struct {
	Name          string   `json:"name"`
	Neighborhoods []string `json:"neighborhoods"`
	FeeCents      int      `json:"feeCents"`
	Enabled       bool     `json:"enabled"`
}

type StoreSettings struct {
	WeeklyHours           map[string]OperatingHours `json:"weeklyHours"`
	Story                 StoreStory                `json:"story"`
	WhatsAppNumber        string                    `json:"whatsAppNumber"`
	ManualOverride        *bool                     `json:"manualOverride"`
	DeliveryOriginAddress string                    `json:"deliveryOriginAddress"`
	DeliveryZones         []DeliveryZone            `json:"deliveryZones"`
}

type StoreStatus struct {
	IsOpen         bool          `json:"isOpen"`
	ManualOverride *bool         `json:"manualOverride"`
	TimeZone       string        `json:"timeZone"`
	Settings       StoreSettings `json:"settings"`
}

var weekdayKeys = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

var neighborhoodDiacritics = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a",
	"é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "õ", "ú", "u", "ç", "c",
)

func NormalizeNeighborhood(value string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(value), " "))
	return neighborhoodDiacritics.Replace(normalized)
}

func FindDeliveryZone(settings StoreSettings, neighborhood string) (DeliveryZone, bool) {
	neighborhood = NormalizeNeighborhood(neighborhood)
	if neighborhood == "" {
		return DeliveryZone{}, false
	}
	for _, zone := range settings.DeliveryZones {
		if !zone.Enabled {
			continue
		}
		for _, coveredNeighborhood := range zone.Neighborhoods {
			if NormalizeNeighborhood(coveredNeighborhood) == neighborhood {
				return zone, true
			}
		}
	}
	return DeliveryZone{}, false
}

func DefaultStoreSettings() StoreSettings {
	return StoreSettings{
		WeeklyHours: map[string]OperatingHours{
			"monday":    {},
			"tuesday":   {Enabled: true, OpensAt: "19:00", ClosesAt: "23:00"},
			"wednesday": {Enabled: true, OpensAt: "19:00", ClosesAt: "23:00"},
			"thursday":  {Enabled: true, OpensAt: "19:00", ClosesAt: "23:00"},
			"friday":    {Enabled: true, OpensAt: "19:00", ClosesAt: "23:00"},
			"saturday":  {Enabled: true, OpensAt: "17:00", ClosesAt: "23:00"},
			"sunday":    {Enabled: true, OpensAt: "17:00", ClosesAt: "23:00"},
		},
		Story: StoreStory{
			Title: "Um intervalo gostoso muda o dia.",
			Body:  "Açaí de verdade, feito com carinho em cada pedido.",
		},
		WhatsAppNumber:        "5521990174473",
		DeliveryOriginAddress: DeliveryOrigin,
		DeliveryZones: []DeliveryZone{
			{
				Name:          "Realengo",
				Neighborhoods: []string{"Realengo"},
				FeeCents:      300,
				Enabled:       true,
			},
			{
				Name:          "Padre Miguel",
				Neighborhoods: []string{"Padre Miguel"},
				FeeCents:      0,
				Enabled:       false,
			},
			{
				Name:          "Bangu",
				Neighborhoods: []string{"Bangu"},
				FeeCents:      0,
				Enabled:       false,
			},
			{
				Name:          "Sulacap",
				Neighborhoods: []string{"Sulacap"},
				FeeCents:      0,
				Enabled:       false,
			},
			{
				Name:          "Magalhães",
				Neighborhoods: []string{"Magalhães Bastos"},
				FeeCents:      0,
				Enabled:       false,
			},
		},
	}
}

func ValidateStoreSettings(settings StoreSettings) error {
	if strings.TrimSpace(settings.Story.Title) == "" || len(settings.Story.Title) > 120 ||
		strings.TrimSpace(settings.Story.Body) == "" || len(settings.Story.Body) > 2000 {
		return ErrInvalidStoreSettings
	}

	whatsAppDigits := NormalizeBrazilianPhone(settings.WhatsAppNumber)
	if strings.HasPrefix(whatsAppDigits, "55") {
		whatsAppDigits = strings.TrimPrefix(whatsAppDigits, "55")
	}
	if !IsValidBrazilianMobilePhone(whatsAppDigits) {
		return ErrInvalidStoreSettings
	}
	if strings.TrimSpace(settings.DeliveryOriginAddress) == "" || len(settings.DeliveryOriginAddress) > 200 || len(settings.DeliveryZones) > 50 {
		return ErrInvalidStoreSettings
	}
	zoneNames := make(map[string]struct{}, len(settings.DeliveryZones))
	neighborhoods := make(map[string]struct{})
	for _, zone := range settings.DeliveryZones {
		zoneName := NormalizeNeighborhood(zone.Name)
		if zoneName == "" || len(zone.Name) > 80 || zone.FeeCents < 0 || zone.FeeCents > 100_000 ||
			len(zone.Neighborhoods) == 0 || len(zone.Neighborhoods) > 50 {
			return ErrInvalidStoreSettings
		}
		if _, exists := zoneNames[zoneName]; exists {
			return ErrInvalidStoreSettings
		}
		zoneNames[zoneName] = struct{}{}
		for _, neighborhood := range zone.Neighborhoods {
			name := NormalizeNeighborhood(neighborhood)
			if name == "" || len(neighborhood) > 80 {
				return ErrInvalidStoreSettings
			}
			if _, exists := neighborhoods[name]; exists {
				return ErrInvalidStoreSettings
			}
			neighborhoods[name] = struct{}{}
		}
	}

	for day, hours := range settings.WeeklyHours {
		if _, validDay := weekdayKeys[day]; !validDay {
			return ErrInvalidStoreSettings
		}
		if !hours.Enabled {
			continue
		}
		opensAt, openErr := time.Parse("15:04", hours.OpensAt)
		closesAt, closeErr := time.Parse("15:04", hours.ClosesAt)
		if openErr != nil || closeErr != nil || !opensAt.Before(closesAt) {
			return ErrInvalidStoreSettings
		}
	}
	return nil
}

func StoreIsOpenAt(settings StoreSettings, now time.Time) bool {
	if settings.ManualOverride != nil {
		return *settings.ManualOverride
	}

	location, err := time.LoadLocation(StoreTimeZone)
	if err != nil {
		return false
	}
	localTime := now.In(location)
	var dayKey string
	for key, weekday := range weekdayKeys {
		if weekday == localTime.Weekday() {
			dayKey = key
			break
		}
	}

	hours, exists := settings.WeeklyHours[dayKey]
	if !exists || !hours.Enabled {
		return false
	}
	currentTime := localTime.Format("15:04")
	return currentTime >= hours.OpensAt && currentTime < hours.ClosesAt
}

func NewStoreStatus(settings StoreSettings, now time.Time) StoreStatus {
	return StoreStatus{
		IsOpen:         StoreIsOpenAt(settings, now),
		ManualOverride: settings.ManualOverride,
		TimeZone:       StoreTimeZone,
		Settings:       settings,
	}
}
