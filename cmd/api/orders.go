package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type createOrderRequest struct {
	Customer            orderCustomer `json:"customer"`
	Acai                orderAcai     `json:"acai"`
	Delivery            orderDelivery `json:"delivery"`
	Payment             orderPayment  `json:"payment"`
	Notes               string        `json:"notes"`
	EstimatedTotalCents int           `json:"estimatedTotalCents" binding:"required,gt=0"`
}

type orderCustomer struct {
	Name  string `json:"name" binding:"required"`
	Phone string `json:"phone" binding:"required"`
}

type orderAcai struct {
	FlavorID            string   `json:"flavorId" binding:"required"`
	SizeID              string   `json:"sizeId" binding:"required"`
	ComboID             string   `json:"comboId"`
	ToppingIDs          []string `json:"toppingIds"`
	SauceID             string   `json:"sauceId"`
	CondimentPositionID string   `json:"condimentPositionId"`
	FruitIDs            []string `json:"fruitIds"`
	ExtraIDs            []string `json:"extraIds"`
}

type orderDelivery struct {
	PostalCode   string `json:"postalCode" binding:"required"`
	Street       string `json:"street" binding:"required"`
	Number       string `json:"number" binding:"required"`
	Neighborhood string `json:"neighborhood" binding:"required"`
	Complement   string `json:"complement"`
	Reference    string `json:"reference"`
}

type orderPayment struct {
	Method         string `json:"method" binding:"required,oneof=pix cash card"`
	NeedsChange    bool   `json:"needsChange"`
	ChangeForCents int    `json:"changeForCents" binding:"gte=0"`
}

type orderListFilter struct {
	Status string
	Search string
	Date   string
	Page   int
	Limit  int
}

type storedOrder struct {
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

type createdOrder struct {
	ID          string
	OrderNumber int
	OrderDate   string
}

type paginatedOrders struct {
	Orders []storedOrder `json:"orders"`
	Page   int           `json:"page"`
	Limit  int           `json:"limit"`
	Total  int64         `json:"total"`
}

type orderStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type dashboardRecentOrder struct {
	ID                  string    `json:"id"`
	OrderNumber         int       `json:"orderNumber"`
	OrderDate           string    `json:"orderDate"`
	Status              string    `json:"status"`
	CustomerName        string    `json:"customerName"`
	EstimatedTotalCents int       `json:"estimatedTotalCents"`
	CreatedAt           time.Time `json:"createdAt"`
}

type dashboardData struct {
	Date               string                 `json:"date"`
	OrdersToday        int64                  `json:"ordersToday"`
	WaitingPreparation int64                  `json:"waitingPreparation"`
	InProduction       int64                  `json:"inProduction"`
	CompletedToday     int64                  `json:"completedToday"`
	StatusCounts       []orderStatusCount     `json:"statusCounts"`
	RecentOrders       []dashboardRecentOrder `json:"recentOrders"`
}

type updateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

var validOrderStatuses = [...]string{
	"received",
	"preparing",
	"ready",
	"out_for_delivery",
	"delivered",
	"completed",
}

func getDashboard(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		date := context.Query("date")
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid dashboard date"})
				return
			}
		}

		result, err := store.Dashboard(context.Request.Context(), date)
		if err != nil {
			log.Printf("get admin dashboard: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load dashboard"})
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func listOrders(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		status := context.Query("status")
		if status != "" && !isValidOrderStatus(status) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order status"})
			return
		}

		page, ok := parsePositiveQuery(context.Query("page"), 1, 1_000_000)
		if !ok {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
		limit, ok := parsePositiveQuery(context.Query("limit"), 20, 100)
		if !ok {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}

		search := strings.TrimSpace(context.Query("search"))
		if len(search) > 100 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "search is too long"})
			return
		}
		date := context.Query("date")
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order date"})
				return
			}
		}

		result, err := store.List(context.Request.Context(), orderListFilter{
			Status: status,
			Search: search,
			Date:   date,
			Page:   page,
			Limit:  limit,
		})
		if err != nil {
			log.Printf("list orders: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not list orders"})
			return
		}

		context.JSON(http.StatusOK, result)
	}
}

func parsePositiveQuery(raw string, fallback, maximum int) (int, bool) {
	if raw == "" {
		return fallback, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maximum {
		return 0, false
	}
	return value, true
}

func isValidOrderStatus(status string) bool {
	for _, validStatus := range validOrderStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

func updateOrderStatus(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		orderID := context.Param("orderId")
		if !isValidUUID(orderID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		var request updateOrderStatusRequest
		if err := context.ShouldBindJSON(&request); err != nil || !isValidOrderStatus(request.Status) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order status"})
			return
		}

		updated, err := store.UpdateStatus(context.Request.Context(), orderID, request.Status)
		if err != nil {
			log.Printf("update order status: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update order status"})
			return
		}
		if !updated {
			context.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}

		context.JSON(http.StatusOK, gin.H{"id": orderID, "status": request.Status})
	}
}

func isValidUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

func createOrder(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createOrderRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order payload"})
			return
		}

		created, err := store.Create(context.Request.Context(), request)
		if err != nil {
			log.Printf("persist order: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not persist order"})
			return
		}

		context.JSON(http.StatusAccepted, gin.H{
			"status":      "received",
			"persisted":   true,
			"orderId":     created.ID,
			"orderNumber": created.OrderNumber,
			"orderDate":   created.OrderDate,
		})
	}
}
