package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

type testOrderStore struct {
	order           createOrderRequest
	created         createdOrder
	err             error
	called          bool
	listResult      paginatedOrders
	listFilter      orderListFilter
	listErr         error
	listCalled      bool
	updatedID       string
	updatedStatus   string
	updateErr       error
	updateFound     bool
	updateCalled    bool
	dashboard       dashboardData
	dashboardDate   string
	dashboardErr    error
	dashboardCalled bool
	catalog         catalogData
	settings        domain.StoreSettings
	settingsErr     error
	createdItem     catalogItemRecord
	updatedItem     catalogItemRecord
	itemFound       bool
	itemArchived    bool
	createdCombo    catalogComboRecord
	updatedCombo    catalogComboRecord
	comboFound      bool
	comboArchived   bool
}

func (store *testOrderStore) Create(_ context.Context, request createOrderRequest) (createdOrder, error) {
	store.order = request
	store.called = true
	return store.created, store.err
}

func (store *testOrderStore) List(_ context.Context, filter orderListFilter) (paginatedOrders, error) {
	store.listFilter = filter
	store.listCalled = true
	return store.listResult, store.listErr
}

func (store *testOrderStore) UpdateStatus(_ context.Context, id, status string) (bool, error) {
	store.updatedID = id
	store.updatedStatus = status
	store.updateCalled = true
	return store.updateFound, store.updateErr
}

func (store *testOrderStore) Dashboard(_ context.Context, date string) (dashboardData, error) {
	store.dashboardDate = date
	store.dashboardCalled = true
	return store.dashboard, store.dashboardErr
}

func (store *testOrderStore) Catalog(_ context.Context) (catalogData, error) {
	return store.catalog, nil
}

func (store *testOrderStore) StoreSettings(_ context.Context) (domain.StoreSettings, error) {
	if store.settingsErr != nil {
		return domain.StoreSettings{}, store.settingsErr
	}
	if store.settings.WeeklyHours == nil {
		store.settings = domain.DefaultStoreSettings()
		open := true
		store.settings.ManualOverride = &open
	}
	return store.settings, nil
}

func (store *testOrderStore) SaveStoreSettings(_ context.Context, settings domain.StoreSettings) error {
	store.settings = settings
	return nil
}

func (store *testOrderStore) CreateCatalogItem(_ context.Context, _ createCatalogItemRequest, _ string, _ bool) (catalogItemRecord, error) {
	return store.createdItem, nil
}

func (store *testOrderStore) UpdateCatalogItem(_ context.Context, _ string, _ updateCatalogItemRequest) (catalogItemRecord, bool, error) {
	return store.updatedItem, store.itemFound, nil
}

func (store *testOrderStore) ArchiveCatalogItem(_ context.Context, _ string) (bool, error) {
	return store.itemArchived, nil
}

func (store *testOrderStore) CreateCatalogCombo(_ context.Context, _ createCatalogComboRequest, _ string, _ bool) (catalogComboRecord, error) {
	return store.createdCombo, nil
}

func (store *testOrderStore) UpdateCatalogCombo(_ context.Context, _ string, _ updateCatalogComboRequest) (catalogComboRecord, bool, error) {
	return store.updatedCombo, store.comboFound, nil
}

func (store *testOrderStore) ArchiveCatalogCombo(_ context.Context, _ string) (bool, error) {
	return store.comboArchived, nil
}

func newRouter(repository application.Repository) *gin.Engine {
	return NewRouter(application.NewService(repository), os.Getenv("ADMIN_API_KEY"))
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}

func TestPublicStoreStatusRespectsManualOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	closed := false
	store := &testOrderStore{settings: domain.DefaultStoreSettings()}
	store.settings.ManualOverride = &closed
	request := httptest.NewRequest(http.MethodGet, "/api/v1/store/status", nil)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var status domain.StoreStatus
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode store status: %v", err)
	}
	if status.IsOpen || status.ManualOverride == nil || *status.ManualOverride {
		t.Fatalf("expected manually closed status, got %+v", status)
	}
}

func TestAdminCanSetStoreManualOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{}
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/settings/override", strings.NewReader(`{"manualOverride":false}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK || store.settings.ManualOverride == nil || *store.settings.ManualOverride {
		t.Fatalf("expected manual closed override, got status %d and settings %+v", response.Code, store.settings)
	}
}

func TestAdminDashboardReturnsAggregatesForRequestedDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{
		dashboard: dashboardData{
			Date:               "2026-09-29",
			OrdersToday:        3,
			WaitingPreparation: 1,
			InProduction:       1,
			CompletedToday:     1,
			StatusCounts: []orderStatusCount{
				{Status: "received", Count: 1},
				{Status: "preparing", Count: 1},
				{Status: "completed", Count: 1},
			},
			RecentOrders: []dashboardRecentOrder{{
				ID:                  "order-1",
				OrderNumber:         3,
				OrderDate:           "2026-09-29",
				Status:              "completed",
				CustomerName:        "Ana Silva",
				EstimatedTotalCents: 1990,
			}},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard?date=2026-09-29", nil)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var body dashboardData
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.OrdersToday != 3 || body.WaitingPreparation != 1 || body.InProduction != 1 || body.CompletedToday != 1 {
		t.Fatalf("unexpected dashboard metrics: %+v", body)
	}
	if !store.dashboardCalled || body.Date != store.dashboardDate || len(body.RecentOrders) != 1 || body.RecentOrders[0].OrderNumber != 3 {
		t.Fatalf("unexpected dashboard date/recent orders: %+v", body)
	}
}

func TestAdminCatalogReturnsItemsCombosAndRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{catalog: catalogData{
		Items:  []catalogItemRecord{{ID: "item-1", ItemKey: "topping-pacoca", Kind: "topping", Name: "Paçoca", PriceCents: 100, Available: true}},
		Combos: []catalogComboRecord{{ID: "combo-1", ComboKey: "combo-special", Name: "Combo Especial", Items: []catalogComboItemRecord{{ItemID: "item-1", Name: "Paçoca", Quantity: 1}}}},
		Rules:  []catalogRuleRecord{{Key: "delivery_fee_cents", Value: float64(300)}},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/catalog", nil)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var body catalogData
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || len(body.Combos) != 1 || len(body.Combos[0].Items) != 1 || len(body.Rules) != 1 {
		t.Fatalf("unexpected catalog response: %+v", body)
	}
}

func TestCreateCatalogItemValidatesAndReturnsItem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{createdItem: catalogItemRecord{
		ID: "item-1", ItemKey: "topping-new-item", Kind: "topping", Name: "Leite Ninho", PriceCents: 150, Available: true,
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/catalog/items", strings.NewReader(`{"kind":"topping","name":" Leite Ninho ","priceCents":150,"sortOrder":10}`))
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var body catalogItemRecord
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Leite Ninho" || body.PriceCents != 150 || !strings.HasPrefix(body.ItemKey, "topping-") {
		t.Fatalf("unexpected catalog item response: %+v", body)
	}
}

func TestCreateCatalogComboRejectsDuplicateItemSelections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	itemID := "a4f535aa-8c2b-4f0f-9c31-783061cc7201"
	requestBody := `{"name":"Combo Paçoca","sizeItemId":"a4f535aa-8c2b-4f0f-9c31-783061cc7202","priceCents":1990,"items":[{"itemId":"` + itemID + `","quantity":1},{"itemId":"` + itemID + `","quantity":1}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/catalog/combos", strings.NewReader(requestBody))
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(&testOrderStore{}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestAdminOrdersReturnsPaginatedOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{
		listResult: paginatedOrders{
			Orders: []storedOrder{{
				ID:                  "order-123",
				OrderNumber:         4,
				OrderDate:           "2026-09-28",
				Status:              "received",
				CustomerName:        "Ana Silva",
				CustomerPhone:       "21999990000",
				EstimatedTotalCents: 1990,
				OrderData:           json.RawMessage(`{"notes":"sem granola"}`),
			}},
			Page:  2,
			Limit: 10,
			Total: 21,
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders?status=received&search=Ana&date=2026-09-28&page=2&limit=10", nil)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var body paginatedOrders
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 21 || body.Page != 2 || body.Limit != 10 || len(body.Orders) != 1 {
		t.Fatalf("unexpected list response: %+v", body)
	}
	if body.Orders[0].CustomerName != "Ana Silva" || string(body.Orders[0].OrderData) != `{"notes":"sem granola"}` {
		t.Fatalf("unexpected order: %+v", body.Orders[0])
	}
	if body.Orders[0].OrderNumber != 4 || body.Orders[0].OrderDate != "2026-09-28" {
		t.Fatalf("unexpected daily order number: %+v", body.Orders[0])
	}
	if !store.listCalled || store.listFilter != (orderListFilter{Status: "received", Search: "Ana", Date: "2026-09-28", Page: 2, Limit: 10}) {
		t.Fatalf("unexpected store filter: %+v", store.listFilter)
	}
}

func TestAdminOrdersRejectsInvalidFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	for _, query := range []string{"?status=unknown", "?page=0", "?limit=101", "?date=2026-02-30"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders"+query, nil)
		request.Header.Set("Authorization", "Bearer "+apiKey)
		response := httptest.NewRecorder()
		store := &testOrderStore{}

		newRouter(store).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for %s, got %d", http.StatusBadRequest, query, response.Code)
		}
		if store.listCalled {
			t.Errorf("store should not be called for invalid filter %s", query)
		}
	}
}

func TestAdminOrderStatusCanBeUpdated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	orderID := "a4f535aa-8c2b-4f0f-9c31-783061cc7201"
	store := &testOrderStore{updateFound: true}
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID, strings.NewReader(`{"status":"preparing"}`))
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if !store.updateCalled || store.updatedID != orderID || store.updatedStatus != "preparing" {
		t.Fatalf("unexpected status update: %+v", store)
	}
}

func TestAdminOrderStatusRejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	for _, testCase := range []struct {
		orderID string
		body    string
	}{
		{orderID: "invalid-id", body: `{"status":"preparing"}`},
		{orderID: "a4f535aa-8c2b-4f0f-9c31-783061cc7201", body: `{"status":"unknown"}`},
	} {
		request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+testCase.orderID, strings.NewReader(testCase.body))
		request.Header.Set("Authorization", "Bearer "+apiKey)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		store := &testOrderStore{}

		newRouter(store).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d for %s", http.StatusBadRequest, response.Code, testCase.orderID)
		}
		if store.updateCalled {
			t.Errorf("store should not be called for invalid update %s", testCase.orderID)
		}
	}
}

func TestAdminHealthRequiresConfiguredAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ADMIN_API_KEY", "")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/health", nil)
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d when API key is missing, got %d", http.StatusServiceUnavailable, response.Code)
	}
}

func TestAdminHealthRejectsInvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ADMIN_API_KEY", "test-admin-api-key-with-at-least-32-characters")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/health", nil)
	request.Header.Set("Authorization", "Bearer incorrect-key")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestAdminHealthAcceptsMatchingAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/health", nil)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
}

func TestCORSPreflightForLocalFrontend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf("expected POST to be allowed, got %q", got)
	}
}

func TestCORSAllowsDeployedFrontendByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://jadesacai.vercel.app")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://jadesacai.vercel.app" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}
}

func TestCORSUsesConfiguredOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://jade.example, https://admin.jade.example")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://admin.jade.example")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.jade.example" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}
}

func TestCORSDoesNotAllowUnlistedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://unlisted.example")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}

func TestMenuCombosEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{catalog: catalogData{
		Items: []catalogItemRecord{
			{ID: "size-300-id", ItemKey: "size-300", Kind: "size", Name: "300 ml", Available: true},
			{ID: "size-500-id", ItemKey: "size-500", Kind: "size", Name: "500 ml", Available: true},
			{ID: "size-770-id", ItemKey: "size-770", Kind: "size", Name: "770 ml", Available: true},
			{ID: "size-1000-id", ItemKey: "size-1000", Kind: "size", Name: "Marmita", Available: true},
		},
		Combos: []catalogComboRecord{
			{ID: "combo-300-id", ComboKey: "combo-300", Name: "Combo 300 ml", SizeItemID: "size-300-id", SizeName: "300 ml", PriceCents: 1290, IncludedToppings: 3, IncludedExtras: 1, Available: true},
			{ID: "combo-500-id", ComboKey: "combo-500", Name: "Combo 500 ml", SizeItemID: "size-500-id", SizeName: "500 ml", PriceCents: 1690, IncludedToppings: 3, IncludedFruits: 1, IncludedExtras: 1, Available: true},
			{ID: "combo-770-id", ComboKey: "combo-770", Name: "Combo 770 ml", SizeItemID: "size-770-id", SizeName: "770 ml", PriceCents: 1990, IncludedToppings: 5, IncludedFruits: 1, IncludedExtras: 1, Available: true},
			{ID: "combo-marmita-id", ComboKey: "combo-marmita", Name: "Combo Marmita", SizeItemID: "size-1000-id", SizeName: "Marmita", PriceCents: 2890, IncludedToppings: 6, IncludedFruits: 2, IncludedExtras: 1, Available: true},
		},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu/combos", nil)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Combos []publicMenuCombo `json:"combos"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Combos) != 4 {
		t.Fatalf("expected 4 combos, got %d", len(body.Combos))
	}
	if body.Combos[0].Key != "combo-300" || body.Combos[0].PriceCents != 1290 {
		t.Errorf("unexpected first combo: %+v", body.Combos[0])
	}
	if body.Combos[3].Key != "combo-marmita" || body.Combos[3].PriceCents != 2890 {
		t.Errorf("unexpected last combo: %+v", body.Combos[3])
	}
}

func TestPublicMenuCatalogExcludesUnavailableRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{catalog: catalogData{
		Items: []catalogItemRecord{
			{ID: "size-id", ItemKey: "size-500", Kind: "size", Name: "500 ml", Available: true},
			{ID: "active-item-id", ItemKey: "topping-pacoca", Kind: "topping", Name: "Paçoca", Available: true},
			{ID: "paused-item-id", ItemKey: "topping-paused", Kind: "topping", Name: "Pausado", Available: false},
			{ID: "archived-item-id", ItemKey: "topping-archived", Kind: "topping", Name: "Arquivado", Available: false, DeletedAt: ptr("2026-09-29T12:00:00Z")},
		},
		Combos: []catalogComboRecord{
			{ID: "active-combo-id", ComboKey: "combo-active", Name: "Combo ativo", SizeItemID: "size-id", SizeName: "500 ml", Available: true},
			{ID: "paused-combo-id", ComboKey: "combo-paused", Name: "Combo pausado", SizeItemID: "size-id", SizeName: "500 ml", Available: false},
		},
		Rules: []catalogRuleRecord{{Key: "delivery_fee_cents", Value: float64(300)}},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu/catalog", nil)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var body publicMenuCatalog
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 2 || len(body.Combos) != 1 || body.Combos[0].Key != "combo-active" {
		t.Fatalf("public catalog included unavailable records: %+v", body)
	}
	if body.Rules["delivery_fee_cents"] != float64(300) {
		t.Fatalf("unexpected public rules: %+v", body.Rules)
	}
}

func TestPublicMenuCatalogIncludesComboCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{catalog: catalogData{
		Items: []catalogItemRecord{{
			ID: "size-id", ItemKey: "size-500", Kind: "size", Name: "500 ml", Available: true,
		}},
		Combos: []catalogComboRecord{{
			ID: "gourmet-combo-id", ComboKey: "combo-gourmet", Category: "gourmet",
			Name: "Combo Gourmet", SizeItemID: "size-id", SizeName: "500 ml", Available: true,
		}},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu/catalog", nil)
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var body publicMenuCatalog
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Combos) != 1 || body.Combos[0].Category != "gourmet" {
		t.Fatalf("expected Gourmet category in public catalog, got %+v", body.Combos)
	}
}

func ptr(value string) *string {
	return &value
}

func TestCreateOrderEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{created: createdOrder{ID: "order-123", OrderNumber: 7, OrderDate: "2026-09-29"}}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"(21) 99999-0000"},"acai":{"flavorId":"banana","sizeId":"500","comboId":"combo-500","toppingIds":["pacoca"],"sauceId":"chocolate","condimentPositionId":"bottom","fruitIds":["banana"],"extraIds":["nutella"]},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro","complement":"","reference":""},"payment":{"method":"pix","needsChange":false,"changeForCents":0},"notes":"","estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}

	var body struct {
		Status      string `json:"status"`
		Persisted   bool   `json:"persisted"`
		OrderID     string `json:"orderId"`
		OrderNumber int    `json:"orderNumber"`
		OrderDate   string `json:"orderDate"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "received" || !body.Persisted || body.OrderID != store.created.ID || body.OrderNumber != 7 || body.OrderDate != "2026-09-29" {
		t.Fatalf("unexpected response: %+v", body)
	}
	if !store.called || store.order.EstimatedTotalCents != 1990 {
		t.Fatalf("expected order to be persisted, got %+v", store)
	}
}

func TestCreateOrderAcceptsMultipleCartItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{created: createdOrder{ID: "order-456", OrderNumber: 8, OrderDate: "2026-09-30"}}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500"},"items":[{"id":"line-1","name":"Combo 500 ml","description":"Açaí de banana · 500 ml · Paçoca","acai":{"flavorId":"banana","sizeId":"500","comboId":"combo-500","toppingIds":["pacoca"],"sauceId":"none","condimentPositionId":"bottom","fruitIds":[],"extraIds":[]},"estimatedSubtotalCents":1690},{"id":"line-2","name":"Açaí livre 300 ml","description":"Açaí de morango · 300 ml · Banana","acai":{"flavorId":"morango","sizeId":"300","comboId":"","toppingIds":[],"sauceId":"chocolate","condimentPositionId":"top","fruitIds":["banana"],"extraIds":[]},"estimatedSubtotalCents":1390}],"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro"},"payment":{"method":"pix"},"notes":"","estimatedTotalCents":3380}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}
	if !store.called || len(store.order.Items) != 2 || store.order.Items[1].Name != "Açaí livre 300 ml" || store.order.Items[1].Description != "Açaí de morango · 300 ml · Banana" {
		t.Fatalf("expected both cart items to be persisted, got %+v", store.order.Items)
	}
}

func TestCreateOrderReturnsErrorWhenPersistenceFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{err: errors.New("database unavailable")}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500"},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro"},"payment":{"method":"pix"},"estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestCreateOrderRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, requestBody := range []string{`{`, `{}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		newRouter(nil).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for %q, got %d", http.StatusBadRequest, requestBody, response.Code)
		}
	}
}

func TestCreateOrderRejectsInvalidPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"(21) 29999-0000"},"acai":{"flavorId":"banana","sizeId":"500"},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro"},"payment":{"method":"pix"},"estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || store.called {
		t.Fatalf("expected invalid phone to be rejected before persistence, got %d", response.Code)
	}
}
