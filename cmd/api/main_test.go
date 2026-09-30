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
	order  createOrderRequest
	id     string
	err    error
	called bool
}

func (store *testOrderStore) Create(_ context.Context, request createOrderRequest) (string, error) {
	store.order = request
	store.called = true
	return store.id, store.err
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
	store := &testOrderStore{id: "order-123"}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500","comboId":"combo-500","toppingIds":["pacoca"],"sauceId":"chocolate","condimentPositionId":"bottom","fruitIds":["banana"],"extraIds":["nutella"]},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro","complement":"","reference":""},"payment":{"method":"pix","needsChange":false,"changeForCents":0},"notes":"","estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}

	var body struct {
		Status    string `json:"status"`
		Persisted bool   `json:"persisted"`
		OrderID   string `json:"orderId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "received" || !body.Persisted || body.OrderID != store.id {
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
