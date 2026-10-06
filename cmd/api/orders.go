package main

import (
	"encoding/hex"
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

func updateOrderStatus(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		orderID := context.Param("orderId")
		if !isValidUUID(orderID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}

		var request updateOrderStatusRequest
		if err := context.ShouldBindJSON(&request); err != nil || !domain.IsValidOrderStatus(request.Status) {
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

func createOrder(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createOrderRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order payload"})
			return
		}
		for _, item := range request.Items {
			if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order item"})
				return
			}
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
