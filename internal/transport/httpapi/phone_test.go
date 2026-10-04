package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateOrderNormalizesFormattedPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &testOrderStore{}
	requestBody := `{"customer":{"name":"Ana Silva","phone":"(21) 99999-0000"},"acai":{"flavorId":"banana","sizeId":"500"},"delivery":{"postalCode":"20000-000","street":"Rua Jade","number":"10","neighborhood":"Centro"},"payment":{"method":"pix"},"estimatedTotalCents":1990}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newRouter(store).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}
	if store.order.Customer.Phone != "21999990000" {
		t.Fatalf("expected phone digits to be normalized, got %q", store.order.Customer.Phone)
	}
}
