package httpapi

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/domain"
)

type publicMenuItem struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	SortOrder  int    `json:"sortOrder"`
}

type publicMenuCombo struct {
	ID               string                `json:"id"`
	Key              string                `json:"key"`
	Category         string                `json:"category"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	SizeID           string                `json:"sizeId"`
	Size             string                `json:"size"`
	PriceCents       int                   `json:"priceCents"`
	IncludedToppings int                   `json:"includedToppings"`
	IncludedFruits   int                   `json:"includedFruits"`
	IncludedExtras   int                   `json:"includedExtras"`
	Tag              string                `json:"tag"`
	ImageURL         string                `json:"image"`
	ImageAlt         string                `json:"imageAlt"`
	Items            []publicMenuComboItem `json:"items"`
}

type publicMenuComboItem struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type publicMenuCatalog struct {
	Items  []publicMenuItem  `json:"items"`
	Combos []publicMenuCombo `json:"combos"`
	Rules  map[string]any    `json:"rules"`
}

func getMenuCatalog(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		catalog, err := store.Catalog(context.Request.Context())
		if err != nil {
			log.Printf("load public menu catalog: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load menu catalog"})
			return
		}
		context.JSON(http.StatusOK, buildPublicMenuCatalog(catalog))
	}
}

func getMenuCombos(store *application.Service) gin.HandlerFunc {
	return func(context *gin.Context) {
		catalog, err := store.Catalog(context.Request.Context())
		if err != nil {
			log.Printf("load public menu combos: %v", err)
			context.JSON(http.StatusInternalServerError, gin.H{"error": "could not load menu combos"})
			return
		}
		context.JSON(http.StatusOK, gin.H{"combos": buildPublicMenuCatalog(catalog).Combos})
	}
}

func buildPublicMenuCatalog(catalog catalogData) publicMenuCatalog {
	result := publicMenuCatalog{
		Items:  make([]publicMenuItem, 0),
		Combos: make([]publicMenuCombo, 0),
		Rules:  make(map[string]any, len(catalog.Rules)),
	}
	activeItems := make(map[string]catalogItemRecord)
	activeMenuItems := make(map[string]publicMenuItem)
	for _, item := range catalog.Items {
		if !item.Available || item.DeletedAt != nil {
			continue
		}
		publicItem := publicMenuItem{
			ID:         item.ID,
			Key:        item.ItemKey,
			Kind:       item.Kind,
			Name:       item.Name,
			PriceCents: item.PriceCents,
			SortOrder:  item.SortOrder,
		}
		activeItems[item.ID] = item
		activeMenuItems[item.ID] = publicItem
		result.Items = append(result.Items, publicItem)
	}
	for _, combo := range catalog.Combos {
		if !combo.Available || combo.DeletedAt != nil {
			continue
		}
		size, exists := activeItems[combo.SizeItemID]
		if !exists || size.Kind != string(domain.CatalogSize) {
			continue
		}
		comboItems := make([]publicMenuComboItem, 0, len(combo.Items))
		allComboItemsAvailable := true
		for _, item := range combo.Items {
			publicItem, itemAvailable := activeMenuItems[item.ItemID]
			if !itemAvailable {
				allComboItemsAvailable = false
				break
			}
			comboItems = append(comboItems, publicMenuComboItem{
				ID:       publicItem.ID,
				Kind:     publicItem.Kind,
				Name:     publicItem.Name,
				Quantity: item.Quantity,
			})
		}
		if !allComboItemsAvailable {
			continue
		}
		result.Combos = append(result.Combos, publicMenuCombo{
			ID:               combo.ID,
			Key:              combo.ComboKey,
			Category:         combo.Category,
			Name:             combo.Name,
			Description:      combo.Description,
			SizeID:           size.ID,
			Size:             size.Name,
			PriceCents:       combo.PriceCents,
			IncludedToppings: combo.IncludedToppings,
			IncludedFruits:   combo.IncludedFruits,
			IncludedExtras:   combo.IncludedExtras,
			Tag:              combo.Tag,
			ImageURL:         combo.ImageURL,
			ImageAlt:         combo.ImageAlt,
			Items:            comboItems,
		})
	}
	for _, rule := range catalog.Rules {
		result.Rules[rule.Key] = rule.Value
	}
	return result
}
