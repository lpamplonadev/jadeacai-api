package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

func TestCreateOrderRejectsWhenStoreIsManuallyClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := domain.DefaultStoreSettings()
	closed := false
	settings.ManualOverride = &closed
	store := &testOrderStore{settings: settings}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"21999990000"},"acai":{"flavorId":"banana","sizeId":"500"},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro"},"payment":{"method":"pix"},"estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusConflict || store.called {
		t.Fatalf("expected closed store to reject order before persistence, got status %d", response.Code)
	}
}

func TestAdminCanUpdateGeneralStoreSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKey := "test-admin-api-key-with-at-least-32-characters"
	t.Setenv("ADMIN_API_KEY", apiKey)
	store := &testOrderStore{settings: domain.DefaultStoreSettings()}
	requestBody := `{"weeklyHours":{"monday":{"enabled":false,"opensAt":"","closesAt":""},"tuesday":{"enabled":true,"opensAt":"19:00","closesAt":"23:00"},"wednesday":{"enabled":true,"opensAt":"19:00","closesAt":"23:00"},"thursday":{"enabled":true,"opensAt":"19:00","closesAt":"23:00"},"friday":{"enabled":true,"opensAt":"19:00","closesAt":"23:00"},"saturday":{"enabled":true,"opensAt":"17:00","closesAt":"23:00"},"sunday":{"enabled":true,"opensAt":"17:00","closesAt":"23:00"}},"story":{"title":"Nossa história","body":"Açaí feito com carinho."},"whatsAppNumber":"(21) 99999-0000","manualOverride":null}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/settings", strings.NewReader(requestBody))
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var status domain.StoreStatus
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode store status: %v", err)
	}
	if status.Settings.Story.Title != "Nossa história" || status.Settings.WhatsAppNumber != "5521999990000" {
		t.Fatalf("unexpected saved store settings: %+v", status.Settings)
	}
}
