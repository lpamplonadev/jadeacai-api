package application

import (
	"encoding/json"
	"time"
)

type CreateOrderRequest struct {
	Customer            OrderCustomer      `json:"customer"`
	Acai                OrderAcai          `json:"acai"`
	Items               []OrderLineRequest `json:"items,omitempty"`
	Delivery            OrderDelivery      `json:"delivery"`
	Payment             OrderPayment       `json:"payment"`
	Notes               string             `json:"notes"`
	EstimatedTotalCents int                `json:"estimatedTotalCents"`
}

type OrderLineRequest struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	Acai                   OrderAcai `json:"acai"`
	EstimatedSubtotalCents int       `json:"estimatedSubtotalCents"`
}

type OrderCustomer struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type OrderAcai struct {
	FlavorID            string   `json:"flavorId"`
	SizeID              string   `json:"sizeId"`
	ComboID             string   `json:"comboId"`
	ToppingIDs          []string `json:"toppingIds"`
	SauceID             string   `json:"sauceId"`
	CondimentPositionID string   `json:"condimentPositionId"`
	FruitIDs            []string `json:"fruitIds"`
	ExtraIDs            []string `json:"extraIds"`
}

type OrderDelivery struct {
	PostalCode   string `json:"postalCode"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	Complement   string `json:"complement"`
	Reference    string `json:"reference"`
}

type OrderPayment struct {
	Method         string `json:"method"`
	NeedsChange    bool   `json:"needsChange"`
	ChangeForCents int    `json:"changeForCents"`
}

type OrderListFilter struct {
	Status string
	Search string
	Date   string
	Page   int
	Limit  int
}

type StoredOrder struct {
	ID                  string          `json:"id"`
	OrderNumber         int             `json:"orderNumber"`
	OrderDate           string          `json:"orderDate"`
	Status              string          `json:"status"`
	CustomerName        string          `json:"customerName"`
	CustomerPhone       string          `json:"customerPhone"`
	EstimatedTotalCents int             `json:"estimatedTotalCents"`
	OrderData           json.RawMessage `json:"orderData"`
	CreatedAt           time.Time       `json:"createdAt"`
}

type CreatedOrder struct {
	ID          string
	OrderNumber int
	OrderDate   string
}

type PaginatedOrders struct {
	Orders []StoredOrder `json:"orders"`
	Page   int           `json:"page"`
	Limit  int           `json:"limit"`
	Total  int64         `json:"total"`
}

type OrderStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type DashboardRecentOrder struct {
	ID                  string    `json:"id"`
	OrderNumber         int       `json:"orderNumber"`
	OrderDate           string    `json:"orderDate"`
	Status              string    `json:"status"`
	CustomerName        string    `json:"customerName"`
	EstimatedTotalCents int       `json:"estimatedTotalCents"`
	CreatedAt           time.Time `json:"createdAt"`
}

type DashboardData struct {
	Date               string                 `json:"date"`
	OrdersToday        int64                  `json:"ordersToday"`
	WaitingPreparation int64                  `json:"waitingPreparation"`
	InProduction       int64                  `json:"inProduction"`
	CompletedToday     int64                  `json:"completedToday"`
	StatusCounts       []OrderStatusCount     `json:"statusCounts"`
	RecentOrders       []DashboardRecentOrder `json:"recentOrders"`
}

type CatalogItemRecord struct {
	ID         string  `json:"id"`
	ItemKey    string  `json:"itemKey"`
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	PriceCents int     `json:"priceCents"`
	Available  bool    `json:"available"`
	SortOrder  int     `json:"sortOrder"`
	DeletedAt  *string `json:"deletedAt"`
}

type CatalogRuleRecord struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type CatalogComboItemRecord struct {
	ItemID    string `json:"itemId"`
	ItemKey   string `json:"itemKey"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	Available bool   `json:"available"`
}

type CatalogComboRecord struct {
	ID               string                   `json:"id"`
	ComboKey         string                   `json:"comboKey"`
	Name             string                   `json:"name"`
	SizeItemID       string                   `json:"sizeItemId"`
	SizeName         string                   `json:"sizeName"`
	PriceCents       int                      `json:"priceCents"`
	IncludedToppings int                      `json:"includedToppings"`
	IncludedFruits   int                      `json:"includedFruits"`
	IncludedExtras   int                      `json:"includedExtras"`
	Tag              string                   `json:"tag"`
	ImageURL         string                   `json:"imageUrl"`
	ImageAlt         string                   `json:"imageAlt"`
	Available        bool                     `json:"available"`
	SortOrder        int                      `json:"sortOrder"`
	DeletedAt        *string                  `json:"deletedAt"`
	Items            []CatalogComboItemRecord `json:"items"`
}

type CatalogData struct {
	Items  []CatalogItemRecord  `json:"items"`
	Combos []CatalogComboRecord `json:"combos"`
	Rules  []CatalogRuleRecord  `json:"rules"`
}

type CreateCatalogItemRequest struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	Available  *bool  `json:"available"`
	SortOrder  int    `json:"sortOrder"`
}

type UpdateCatalogItemRequest struct {
	Name       *string `json:"name"`
	PriceCents *int    `json:"priceCents"`
	Available  *bool   `json:"available"`
	SortOrder  *int    `json:"sortOrder"`
}

type CatalogComboItemInput struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type CreateCatalogComboRequest struct {
	Name             string                  `json:"name"`
	SizeItemID       string                  `json:"sizeItemId"`
	PriceCents       int                     `json:"priceCents"`
	IncludedToppings int                     `json:"includedToppings"`
	IncludedFruits   int                     `json:"includedFruits"`
	IncludedExtras   int                     `json:"includedExtras"`
	Tag              string                  `json:"tag"`
	ImageURL         string                  `json:"imageUrl"`
	ImageAlt         string                  `json:"imageAlt"`
	Available        *bool                   `json:"available"`
	SortOrder        int                     `json:"sortOrder"`
	Items            []CatalogComboItemInput `json:"items"`
}

type UpdateCatalogComboRequest struct {
	Name             *string                  `json:"name"`
	SizeItemID       *string                  `json:"sizeItemId"`
	PriceCents       *int                     `json:"priceCents"`
	IncludedToppings *int                     `json:"includedToppings"`
	IncludedFruits   *int                     `json:"includedFruits"`
	IncludedExtras   *int                     `json:"includedExtras"`
	Tag              *string                  `json:"tag"`
	ImageURL         *string                  `json:"imageUrl"`
	ImageAlt         *string                  `json:"imageAlt"`
	Available        *bool                    `json:"available"`
	SortOrder        *int                     `json:"sortOrder"`
	Items            *[]CatalogComboItemInput `json:"items"`
}
