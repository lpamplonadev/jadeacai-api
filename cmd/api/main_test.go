package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type testOrderStore struct {
	order         createOrderRequest
	created       createdOrder
	err           error
	called        bool
	listResult    paginatedOrders
	listFilter    orderListFilter
	listErr       error
	listCalled    bool
	updatedID     string
	updatedStatus string
	updateErr     error
	updateFound   bool
	updateCalled  bool
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

func TestOpenOrderStoreRequiresDatabaseURL(t *testing.T) {
	if _, err := openOrderStore(""); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
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
	request.Header.Set("Origin", "https://jadeacai-web.vercel.app")
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://jadeacai-web.vercel.app" {
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
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu/combos", nil)
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Combos []menuCombo `json:"combos"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Combos) != 4 {
		t.Fatalf("expected 4 combos, got %d", len(body.Combos))
	}
	if body.Combos[0].ID != "combo-300" || body.Combos[0].PriceCents != 1290 {
		t.Errorf("unexpected first combo: %+v", body.Combos[0])
	}
	if body.Combos[3].ID != "combo-marmita" || body.Combos[3].PriceCents != 2890 {
		t.Errorf("unexpected last combo: %+v", body.Combos[3])
	}
}

func TestCreateOrderEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{created: createdOrder{ID: "order-123", OrderNumber: 7, OrderDate: "2026-09-29"}}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500","comboId":"combo-500","toppingIds":["pacoca"],"sauceId":"chocolate","condimentPositionId":"bottom","fruitIds":["banana"],"extraIds":["nutella"]},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro","complement":"","reference":""},"payment":{"method":"pix","needsChange":false,"changeForCents":0},"notes":"","estimatedTotalCents":1990}`
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
