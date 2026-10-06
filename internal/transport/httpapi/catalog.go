package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

func getCatalog(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		catalog, err := store.Catalog(context.Request.Context())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load catalog"})
			return
		}
		context.JSON(http.StatusOK, catalog)
	}
}

func createCatalogItem(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createCatalogItemRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item"})
			return
		}
		item, err := store.CreateItem(context.Request.Context(), request)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item"})
			return
		}
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not create catalog item"})
			return
		}
		context.JSON(http.StatusCreated, item)
	}
}

func updateCatalogItem(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		itemID := context.Param("itemId")
		if !domain.IsValidUUID(itemID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item id"})
			return
		}

		var request updateCatalogItemRequest
		if err := context.ShouldBindJSON(&request); err != nil || (request.Name == nil && request.PriceCents == nil && request.Available == nil && request.SortOrder == nil) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "no valid catalog item fields"})
			return
		}
		item, found, err := store.UpdateItem(context.Request.Context(), itemID, request)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item values"})
			return
		}
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update catalog item"})
			return
		}
		if !found {
			context.JSON(http.StatusNotFound, gin.H{"error": "catalog item not found"})
			return
		}
		context.JSON(http.StatusOK, item)
	}
}

func archiveCatalogItem(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		itemID := context.Param("itemId")
		if !domain.IsValidUUID(itemID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item id"})
			return
		}
		archived, err := store.ArchiveCatalogItem(context.Request.Context(), itemID)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not archive catalog item"})
			return
		}
		if !archived {
			context.JSON(http.StatusNotFound, gin.H{"error": "catalog item not found"})
			return
		}
		context.Status(http.StatusNoContent)
	}
}

func createCatalogCombo(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createCatalogComboRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo"})
			return
		}
		combo, err := store.CreateCombo(context.Request.Context(), request)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo"})
			return
		}
		if errors.Is(err, application.ErrCatalogReference) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "combo must reference active catalog items"})
			return
		}
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not create catalog combo"})
			return
		}
		context.JSON(http.StatusCreated, combo)
	}
}

func updateCatalogCombo(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		comboID := context.Param("comboId")
		if !domain.IsValidUUID(comboID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo id"})
			return
		}
		var request updateCatalogComboRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo"})
			return
		}
		combo, found, err := store.UpdateCombo(context.Request.Context(), comboID, request)
		if errors.Is(err, application.ErrInvalidInput) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo values"})
			return
		}
		if errors.Is(err, application.ErrCatalogReference) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "combo must reference active catalog items"})
			return
		}
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not update catalog combo"})
			return
		}
		if !found {
			context.JSON(http.StatusNotFound, gin.H{"error": "catalog combo not found"})
			return
		}
		context.JSON(http.StatusOK, combo)
	}
}

func archiveCatalogCombo(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		comboID := context.Param("comboId")
		if !domain.IsValidUUID(comboID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo id"})
			return
		}
		archived, err := store.ArchiveCatalogCombo(context.Request.Context(), comboID)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not archive catalog combo"})
			return
		}
		if !archived {
			context.JSON(http.StatusNotFound, gin.H{"error": "catalog combo not found"})
			return
		}
		context.Status(http.StatusNoContent)
	}
}

func hasCatalogComboUpdate(request updateCatalogComboRequest) bool {
	return request.Name != nil || request.SizeItemID != nil || request.PriceCents != nil ||
		request.Description != nil || request.IncludedToppings != nil || request.IncludedFruits != nil || request.IncludedExtras != nil ||
		request.Tag != nil || request.ImageURL != nil || request.ImageAlt != nil ||
		request.Available != nil || request.SortOrder != nil || request.Items != nil
}
