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
