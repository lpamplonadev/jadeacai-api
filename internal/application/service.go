package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

var ErrCatalogReference = errors.New("catalog reference is missing, inactive, or has an incompatible type")
var ErrInvalidInput = errors.New("invalid application input")

type Repository interface {
	Create(context.Context, CreateOrderRequest) (CreatedOrder, error)
	StoreSettings(context.Context) (domain.StoreSettings, error)
	SaveStoreSettings(context.Context, domain.StoreSettings) error
	List(context.Context, OrderListFilter) (PaginatedOrders, error)
	Dashboard(context.Context, string) (DashboardData, error)
	UpdateStatus(context.Context, string, string) (bool, error)
	Catalog(context.Context) (CatalogData, error)
	CreateCatalogItem(context.Context, CreateCatalogItemRequest, string, bool) (CatalogItemRecord, error)
	UpdateCatalogItem(context.Context, string, UpdateCatalogItemRequest) (CatalogItemRecord, bool, error)
	ArchiveCatalogItem(context.Context, string) (bool, error)
	CreateCatalogCombo(context.Context, CreateCatalogComboRequest, string, bool) (CatalogComboRecord, error)
	UpdateCatalogCombo(context.Context, string, UpdateCatalogComboRequest) (CatalogComboRecord, bool, error)
	ArchiveCatalogCombo(context.Context, string) (bool, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Create(ctx context.Context, request CreateOrderRequest) (CreatedOrder, error) {
	if !domain.IsValidBrazilianMobilePhone(request.Customer.Phone) {
		return CreatedOrder{}, domain.ErrInvalidPhone
	}
	request.Customer.Phone = domain.NormalizeBrazilianPhone(request.Customer.Phone)
	if request.EstimatedTotalCents <= 0 ||
		strings.TrimSpace(request.Customer.Name) == "" || strings.TrimSpace(request.Customer.Phone) == "" ||
		strings.TrimSpace(request.Acai.FlavorID) == "" || strings.TrimSpace(request.Acai.SizeID) == "" ||
		strings.TrimSpace(request.Delivery.PostalCode) == "" || strings.TrimSpace(request.Delivery.Street) == "" ||
		strings.TrimSpace(request.Delivery.Number) == "" || strings.TrimSpace(request.Delivery.Neighborhood) == "" ||
		(request.Payment.Method != "pix" && request.Payment.Method != "cash" && request.Payment.Method != "card") ||
		request.Payment.ChangeForCents < 0 || len(request.Items) > 100 {
		return CreatedOrder{}, ErrInvalidInput
	}
	for _, item := range request.Items {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" ||
			strings.TrimSpace(item.Acai.FlavorID) == "" || strings.TrimSpace(item.Acai.SizeID) == "" ||
			item.EstimatedSubtotalCents < 0 {
			return CreatedOrder{}, ErrInvalidInput
		}
	}
	status, err := service.StoreStatus(ctx, time.Now())
	if err != nil {
		return CreatedOrder{}, err
	}
	if !status.IsOpen {
		return CreatedOrder{}, domain.ErrStoreClosed
	}
	return service.repository.Create(ctx, request)
}

func (service *Service) StoreSettings(ctx context.Context) (domain.StoreSettings, error) {
	settings, err := service.repository.StoreSettings(ctx)
	if err != nil {
		return domain.StoreSettings{}, err
	}
	if err := domain.ValidateStoreSettings(settings); err != nil {
		return domain.StoreSettings{}, err
	}
	return settings, nil
}

func (service *Service) StoreStatus(ctx context.Context, now time.Time) (domain.StoreStatus, error) {
	settings, err := service.StoreSettings(ctx)
	if err != nil {
		return domain.StoreStatus{}, err
	}
	return domain.NewStoreStatus(settings, now), nil
}

func (service *Service) UpdateStoreSettings(ctx context.Context, settings domain.StoreSettings) error {
	if err := domain.ValidateStoreSettings(settings); err != nil {
		return err
	}
	whatsAppNumber := domain.NormalizeBrazilianPhone(settings.WhatsAppNumber)
	if strings.HasPrefix(whatsAppNumber, "55") {
		whatsAppNumber = strings.TrimPrefix(whatsAppNumber, "55")
	}
	settings.WhatsAppNumber = "55" + whatsAppNumber
	return service.repository.SaveStoreSettings(ctx, settings)
}

func (service *Service) SetStoreOverride(ctx context.Context, override *bool) (domain.StoreSettings, error) {
	settings, err := service.StoreSettings(ctx)
	if err != nil {
		return domain.StoreSettings{}, err
	}
	settings.ManualOverride = override
	if err := service.UpdateStoreSettings(ctx, settings); err != nil {
		return domain.StoreSettings{}, err
	}
	return settings, nil
}

func (service *Service) List(ctx context.Context, filter OrderListFilter) (PaginatedOrders, error) {
	if (filter.Status != "" && !domain.IsValidOrderStatus(filter.Status)) ||
		len(filter.Search) > 100 || filter.Page < 1 || filter.Limit < 1 || filter.Limit > 100 {
		return PaginatedOrders{}, ErrInvalidInput
	}
	if filter.Date != "" {
		if _, err := time.Parse("2006-01-02", filter.Date); err != nil {
			return PaginatedOrders{}, ErrInvalidInput
		}
	}
	return service.repository.List(ctx, filter)
}

func (service *Service) Dashboard(ctx context.Context, date string) (DashboardData, error) {
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return DashboardData{}, ErrInvalidInput
		}
	}
	return service.repository.Dashboard(ctx, date)
}

func (service *Service) UpdateStatus(ctx context.Context, orderID, status string) (bool, error) {
	if !domain.IsValidOrderStatus(status) {
		return false, ErrInvalidInput
	}
	return service.repository.UpdateStatus(ctx, orderID, status)
}

func (service *Service) Catalog(ctx context.Context) (CatalogData, error) {
	return service.repository.Catalog(ctx)
}

func (service *Service) CreateItem(ctx context.Context, request CreateCatalogItemRequest) (CatalogItemRecord, error) {
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len(request.Name) > 120 || !domain.IsCatalogItemKind(request.Kind) || request.PriceCents < 0 || request.SortOrder < 0 {
		return CatalogItemRecord{}, ErrInvalidInput
	}
	itemKey, err := newCatalogKey(request.Kind)
	if err != nil {
		return CatalogItemRecord{}, fmt.Errorf("create catalog item key: %w", err)
	}
	available := request.Available == nil || *request.Available
	return service.repository.CreateCatalogItem(ctx, request, itemKey, available)
}

func (service *Service) UpdateItem(ctx context.Context, itemID string, request UpdateCatalogItemRequest) (CatalogItemRecord, bool, error) {
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" || len(name) > 120 {
			return CatalogItemRecord{}, false, ErrInvalidInput
		}
		request.Name = &name
	}
	if (request.PriceCents != nil && *request.PriceCents < 0) ||
		(request.SortOrder != nil && *request.SortOrder < 0) ||
		(request.Name == nil && request.PriceCents == nil && request.Available == nil && request.SortOrder == nil) {
		return CatalogItemRecord{}, false, ErrInvalidInput
	}
	return service.repository.UpdateCatalogItem(ctx, itemID, request)
}

func (service *Service) ArchiveCatalogItem(ctx context.Context, itemID string) (bool, error) {
	return service.repository.ArchiveCatalogItem(ctx, itemID)
}

func (service *Service) CreateCombo(ctx context.Context, request CreateCatalogComboRequest) (CatalogComboRecord, error) {
	request.Name = strings.TrimSpace(request.Name)
	if err := validateCatalogCombo(request); err != nil {
		return CatalogComboRecord{}, err
	}
	comboKey, err := newCatalogKey("combo")
	if err != nil {
		return CatalogComboRecord{}, fmt.Errorf("create catalog combo key: %w", err)
	}
	available := request.Available == nil || *request.Available
	return service.repository.CreateCatalogCombo(ctx, request, comboKey, available)
}

func (service *Service) UpdateCombo(ctx context.Context, comboID string, request UpdateCatalogComboRequest) (CatalogComboRecord, bool, error) {
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" || len(name) > 120 {
			return CatalogComboRecord{}, false, ErrInvalidInput
		}
		request.Name = &name
	}
	if (request.PriceCents != nil && *request.PriceCents < 0) ||
		(request.IncludedToppings != nil && *request.IncludedToppings < 0) ||
		(request.IncludedFruits != nil && *request.IncludedFruits < 0) ||
		(request.IncludedExtras != nil && *request.IncludedExtras < 0) ||
		(request.SortOrder != nil && *request.SortOrder < 0) ||
		(request.SizeItemID != nil && !domain.IsValidUUID(*request.SizeItemID)) ||
		!hasCatalogComboUpdate(request) {
		return CatalogComboRecord{}, false, ErrInvalidInput
	}
	if request.Items != nil {
		if err := validateComboItems(*request.Items); err != nil {
			return CatalogComboRecord{}, false, err
		}
	}
	return service.repository.UpdateCatalogCombo(ctx, comboID, request)
}

func (service *Service) ArchiveCatalogCombo(ctx context.Context, comboID string) (bool, error) {
	return service.repository.ArchiveCatalogCombo(ctx, comboID)
}

func validateCatalogCombo(request CreateCatalogComboRequest) error {
	if request.Name == "" || len(request.Name) > 120 || !domain.IsValidUUID(request.SizeItemID) ||
		request.PriceCents < 0 || request.IncludedToppings < 0 || request.IncludedFruits < 0 ||
		request.IncludedExtras < 0 || request.SortOrder < 0 {
		return ErrInvalidInput
	}
	return validateComboItems(request.Items)
}

func validateComboItems(items []CatalogComboItemInput) error {
	seen := make(map[string]struct{}, len(items))
	totalQuantity := 0
	for _, item := range items {
		if !domain.IsValidUUID(item.ItemID) || item.Quantity < 1 || item.Quantity > 100 {
			return ErrInvalidInput
		}
		totalQuantity += item.Quantity
		if totalQuantity > 100 {
			return ErrInvalidInput
		}
		if _, exists := seen[item.ItemID]; exists {
			return ErrInvalidInput
		}
		seen[item.ItemID] = struct{}{}
	}
	return nil
}

func hasCatalogComboUpdate(request UpdateCatalogComboRequest) bool {
	return request.Name != nil || request.SizeItemID != nil || request.PriceCents != nil ||
		request.IncludedToppings != nil || request.IncludedFruits != nil || request.IncludedExtras != nil ||
		request.Tag != nil || request.ImageURL != nil || request.ImageAlt != nil ||
		request.Available != nil || request.SortOrder != nil || request.Items != nil
}

func newCatalogKey(prefix string) (string, error) {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(identifier), nil
}
