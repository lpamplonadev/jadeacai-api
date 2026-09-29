package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected response body: %s", response.Body.String())
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

	newRouter().ServeHTTP(response, request)

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

	newRouter().ServeHTTP(response, request)

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

	newRouter().ServeHTTP(response, request)

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

	newRouter().ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}

func TestMenuCombosEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu/combos", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

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
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500","comboId":"combo-500","toppingIds":["pacoca"],"sauceId":"chocolate","condimentPositionId":"bottom","fruitIds":["banana"],"extraIds":["nutella"]},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro","complement":"","reference":""},"payment":{"method":"pix","needsChange":false,"changeForCents":0},"notes":"","estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}

	var body struct {
		Status    string `json:"status"`
		Persisted bool   `json:"persisted"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "received" || body.Persisted {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestCreateOrderRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, requestBody := range []string{`{`, `{}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		newRouter().ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for %q, got %d", http.StatusBadRequest, requestBody, response.Code)
		}
	}
}
