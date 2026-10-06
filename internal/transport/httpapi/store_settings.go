package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

type updateStoreOverrideRequest struct {
	ManualOverride *bool `json:"manualOverride"`
}

func getStoreStatus(service *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		status, err := service.StoreStatus(context.Request.Context(), time.Now())
		if err != nil {
			context.JSON(http.StatusServiceUnavailable, gin.H{"error": "store status unavailable"})
			return
		}
		context.JSON(http.StatusOK, status)
	}
}

func getStoreSettings(service *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		status, err := service.StoreStatus(context.Request.Context(), time.Now())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load store settings"})
			return
		}
		context.JSON(http.StatusOK, status)
	}
}

func updateStoreSettings(service *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var settings domain.StoreSettings
		if err := context.ShouldBindJSON(&settings); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid store settings"})
			return
		}
		if err := service.UpdateStoreSettings(context.Request.Context(), settings); errors.Is(err, domain.ErrInvalidStoreSettings) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid store settings"})
			return
		} else if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not save store settings"})
			return
		}
		updated, err := service.StoreStatus(context.Request.Context(), time.Now())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not reload store settings"})
			return
		}
		context.JSON(http.StatusOK, updated)
	}
}

func updateStoreOverride(service *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request updateStoreOverrideRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid store override"})
			return
		}
		_, err := service.SetStoreOverride(context.Request.Context(), request.ManualOverride)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update store override"})
			return
		}
		status, err := service.StoreStatus(context.Request.Context(), time.Now())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not reload store status"})
			return
		}
		context.JSON(http.StatusOK, status)
	}
}
