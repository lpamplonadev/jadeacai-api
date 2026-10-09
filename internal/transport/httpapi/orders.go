package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

type updateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func getDashboard(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		date := context.Query("date")
		result, err := store.Dashboard(context.Request.Context(), date)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid dashboard date"})
			return
		}
		if err != nil {
			log.Printf("get admin dashboard: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load dashboard"})
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func listOrders(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		status := context.Query("status")
		if status != "" && !domain.IsValidOrderStatus(status) {
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
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order filters"})
			return
		}
		if err != nil {
			log.Printf("list orders: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not list orders"})
			return
		}

		context.JSON(http.StatusOK, result)
	}
}

func trackOrder(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		orderID := context.Param("orderId")
		if !domain.IsValidUUID(orderID) {
			context.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}

		order, found, err := store.TrackOrder(context.Request.Context(), orderID)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if err != nil {
			log.Printf("track order: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load order status"})
			return
		}
		if !found {
			context.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}

		context.Header("Cache-Control", "no-store")
		context.JSON(http.StatusOK, order)
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

func updateOrderStatus(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		orderID := context.Param("orderId")
		if !domain.IsValidUUID(orderID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		var request updateOrderStatusRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order status"})
			return
		}

		updated, err := store.UpdateStatus(context.Request.Context(), orderID, request.Status)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order status"})
			return
		}
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

func createOrder(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createOrderRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order payload"})
			return
		}
		created, err := store.Create(context.Request.Context(), request)
		if errors.Is(err, domain.ErrStoreClosed) {
			context.JSON(http.StatusConflict, gin.H{"error": "store_closed", "message": "A loja está fechada no momento."})
			return
		}
		if errors.Is(err, domain.ErrInvalidPhone) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer phone"})
			return
		}
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order item"})
			return
		}
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
