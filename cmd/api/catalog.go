package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type catalogItemKind string

const (
	catalogFlavor            catalogItemKind = "flavor"
	catalogSize              catalogItemKind = "size"
	catalogTopping           catalogItemKind = "topping"
	catalogSauce             catalogItemKind = "sauce"
	catalogCondimentPosition catalogItemKind = "condiment_position"
	catalogFruit             catalogItemKind = "fruit"
	catalogExtra             catalogItemKind = "extra"
)

var catalogItemKinds = [...]catalogItemKind{
	catalogFlavor,
	catalogSize,
	catalogTopping,
	catalogSauce,
	catalogCondimentPosition,
	catalogFruit,
	catalogExtra,
}

var errCatalogReference = errors.New("catalog reference is missing, inactive, or has an incompatible type")

type catalogItemRecord struct {
	ID         string  `json:"id"`
	ItemKey    string  `json:"itemKey"`
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	PriceCents int     `json:"priceCents"`
	Available  bool    `json:"available"`
	SortOrder  int     `json:"sortOrder"`
	DeletedAt  *string `json:"deletedAt"`
}

type catalogRuleRecord struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type catalogComboItemRecord struct {
	ItemID    string `json:"itemId"`
	ItemKey   string `json:"itemKey"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	Available bool   `json:"available"`
}

type catalogComboRecord struct {
	ID               string                   `json:"id"`
	ComboKey         string                   `json:"comboKey"`
	Name             string                   `json:"name"`
	SizeItemID       string                   `json:"sizeItemId"`
	SizeName         string                   `json:"sizeName"`
	PriceCents       int                      `json:"priceCents"`
	IncludedToppings int                      `json:"includedToppings"`
	IncludedFruits   int                      `json:"includedFruits"`
	IncludedExtras   int                      `json:"includedExtras"`
	Tag              string                   `json:"tag"`
	ImageURL         string                   `json:"imageUrl"`
	ImageAlt         string                   `json:"imageAlt"`
	Available        bool                     `json:"available"`
	SortOrder        int                      `json:"sortOrder"`
	DeletedAt        *string                  `json:"deletedAt"`
	Items            []catalogComboItemRecord `json:"items"`
}

type catalogData struct {
	Items  []catalogItemRecord  `json:"items"`
	Combos []catalogComboRecord `json:"combos"`
	Rules  []catalogRuleRecord  `json:"rules"`
}

type createCatalogItemRequest struct {
	Kind       string `json:"kind" binding:"required"`
	Name       string `json:"name" binding:"required"`
	PriceCents int    `json:"priceCents" binding:"gte=0"`
	Available  *bool  `json:"available"`
	SortOrder  int    `json:"sortOrder" binding:"gte=0"`
}

type updateCatalogItemRequest struct {
	Name       *string `json:"name"`
	PriceCents *int    `json:"priceCents"`
	Available  *bool   `json:"available"`
	SortOrder  *int    `json:"sortOrder"`
}

type catalogComboItemInput struct {
	ItemID   string `json:"itemId" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gte=1,lte=100"`
}

type createCatalogComboRequest struct {
	Name             string                  `json:"name" binding:"required"`
	SizeItemID       string                  `json:"sizeItemId" binding:"required"`
	PriceCents       int                     `json:"priceCents" binding:"gte=0"`
	IncludedToppings int                     `json:"includedToppings" binding:"gte=0"`
	IncludedFruits   int                     `json:"includedFruits" binding:"gte=0"`
	IncludedExtras   int                     `json:"includedExtras" binding:"gte=0"`
	Tag              string                  `json:"tag"`
	ImageURL         string                  `json:"imageUrl"`
	ImageAlt         string                  `json:"imageAlt"`
	Available        *bool                   `json:"available"`
	SortOrder        int                     `json:"sortOrder" binding:"gte=0"`
	Items            []catalogComboItemInput `json:"items"`
}

type updateCatalogComboRequest struct {
	Name             *string                  `json:"name"`
	SizeItemID       *string                  `json:"sizeItemId"`
	PriceCents       *int                     `json:"priceCents"`
	IncludedToppings *int                     `json:"includedToppings"`
	IncludedFruits   *int                     `json:"includedFruits"`
	IncludedExtras   *int                     `json:"includedExtras"`
	Tag              *string                  `json:"tag"`
	ImageURL         *string                  `json:"imageUrl"`
	ImageAlt         *string                  `json:"imageAlt"`
	Available        *bool                    `json:"available"`
	SortOrder        *int                     `json:"sortOrder"`
	Items            *[]catalogComboItemInput `json:"items"`
}

func getCatalog(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		catalog, err := store.Catalog(context.Request.Context())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load catalog"})
			return
		}
		context.JSON(http.StatusOK, catalog)
	}
}

func createCatalogItem(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createCatalogItemRequest
		if err := context.ShouldBindJSON(&request); err != nil || !isValidCatalogItemKind(request.Kind) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item"})
			return
		}
		request.Name = strings.TrimSpace(request.Name)
		if request.Name == "" || len(request.Name) > 120 || request.PriceCents < 0 || request.SortOrder < 0 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item"})
			return
		}
		available := true
		if request.Available != nil {
			available = *request.Available
		}
		itemKey, err := newCatalogKey(request.Kind)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not create catalog item"})
			return
		}

		item, err := store.CreateCatalogItem(context.Request.Context(), request, itemKey, available)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not create catalog item"})
			return
		}
		context.JSON(http.StatusCreated, item)
	}
}

func updateCatalogItem(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		itemID := context.Param("itemId")
		if !isValidUUID(itemID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item id"})
			return
		}

		var request updateCatalogItemRequest
		if err := context.ShouldBindJSON(&request); err != nil || (request.Name == nil && request.PriceCents == nil && request.Available == nil && request.SortOrder == nil) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "no valid catalog item fields"})
			return
		}
		if request.Name != nil {
			name := strings.TrimSpace(*request.Name)
			if name == "" || len(name) > 120 {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item name"})
				return
			}
			request.Name = &name
		}
		if (request.PriceCents != nil && *request.PriceCents < 0) || (request.SortOrder != nil && *request.SortOrder < 0) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog item values"})
			return
		}

		item, found, err := store.UpdateCatalogItem(context.Request.Context(), itemID, request)
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

func archiveCatalogItem(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		itemID := context.Param("itemId")
		if !isValidUUID(itemID) {
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

func createCatalogCombo(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request createCatalogComboRequest
		if err := context.ShouldBindJSON(&request); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo"})
			return
		}
		if err := validateCatalogCombo(request.Name, request.SizeItemID, request.PriceCents, request.IncludedToppings, request.IncludedFruits, request.IncludedExtras, request.SortOrder, request.Items); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		comboKey, err := newCatalogKey("combo")
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not create catalog combo"})
			return
		}
		available := true
		if request.Available != nil {
			available = *request.Available
		}

		combo, err := store.CreateCatalogCombo(context.Request.Context(), request, comboKey, available)
		if errors.Is(err, errCatalogReference) {
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

func updateCatalogCombo(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		comboID := context.Param("comboId")
		if !isValidUUID(comboID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo id"})
			return
		}
		var request updateCatalogComboRequest
		if err := context.ShouldBindJSON(&request); err != nil || !hasCatalogComboUpdate(request) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "no valid catalog combo fields"})
			return
		}
		if request.Name != nil {
			name := strings.TrimSpace(*request.Name)
			if name == "" || len(name) > 120 {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo name"})
				return
			}
			request.Name = &name
		}
		if request.PriceCents != nil && *request.PriceCents < 0 ||
			request.IncludedToppings != nil && *request.IncludedToppings < 0 ||
			request.IncludedFruits != nil && *request.IncludedFruits < 0 ||
			request.IncludedExtras != nil && *request.IncludedExtras < 0 ||
			request.SortOrder != nil && *request.SortOrder < 0 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog combo values"})
			return
		}
		if request.SizeItemID != nil && !isValidUUID(*request.SizeItemID) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid combo size item id"})
			return
		}
		if request.Items != nil {
			if err := validateComboItems(*request.Items); err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		combo, found, err := store.UpdateCatalogCombo(context.Request.Context(), comboID, request)
		if errors.Is(err, errCatalogReference) {
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

func archiveCatalogCombo(store orderStore) gin.HandlerFunc {
	return func(context *gin.Context) {
		comboID := context.Param("comboId")
		if !isValidUUID(comboID) {
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

func validateCatalogCombo(name, sizeItemID string, priceCents, includedToppings, includedFruits, includedExtras, sortOrder int, items []catalogComboItemInput) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 || !isValidUUID(sizeItemID) || priceCents < 0 || includedToppings < 0 || includedFruits < 0 || includedExtras < 0 || sortOrder < 0 {
		return errors.New("invalid catalog combo values")
	}
	return validateComboItems(items)
}

func validateComboItems(items []catalogComboItemInput) error {
	seen := make(map[string]struct{}, len(items))
	totalQuantity := 0
	for _, item := range items {
		if !isValidUUID(item.ItemID) || item.Quantity < 1 || item.Quantity > 100 {
			return errors.New("invalid combo item selection")
		}
		totalQuantity += item.Quantity
		if totalQuantity > 100 {
			return errors.New("combo cannot contain more than 100 items")
		}
		if _, exists := seen[item.ItemID]; exists {
			return errors.New("duplicate combo item selection")
		}
		seen[item.ItemID] = struct{}{}
	}
	return nil
}

func hasCatalogComboUpdate(request updateCatalogComboRequest) bool {
	return request.Name != nil || request.SizeItemID != nil || request.PriceCents != nil ||
		request.IncludedToppings != nil || request.IncludedFruits != nil || request.IncludedExtras != nil ||
		request.Tag != nil || request.ImageURL != nil || request.ImageAlt != nil ||
		request.Available != nil || request.SortOrder != nil || request.Items != nil
}

func isValidCatalogItemKind(kind string) bool {
	for _, validKind := range catalogItemKinds {
		if kind == string(validKind) {
			return true
		}
	}
	return false
}

func newCatalogKey(prefix string) (string, error) {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(identifier), nil
}
