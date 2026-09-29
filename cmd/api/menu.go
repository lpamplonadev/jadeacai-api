package main

import "github.com/gin-gonic/gin"

type menuCombo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Size             string `json:"size"`
	PriceCents       int    `json:"priceCents"`
	IncludedToppings int    `json:"includedToppings"`
	IncludedFruits   int    `json:"includedFruits"`
	IncludedExtras   int    `json:"includedExtras"`
	Tag              string `json:"tag"`
}

var menuCombos = []menuCombo{
	{ID: "combo-300", Name: "Combo 300 ml", Size: "300 ml", PriceCents: 1290, IncludedToppings: 3, IncludedFruits: 0, IncludedExtras: 1, Tag: "Seu primeiro Jade"},
	{ID: "combo-500", Name: "Combo 500 ml", Size: "500 ml", PriceCents: 1690, IncludedToppings: 3, IncludedFruits: 1, IncludedExtras: 1, Tag: "Mais pedido"},
	{ID: "combo-770", Name: "Combo 770 ml", Size: "770 ml", PriceCents: 1990, IncludedToppings: 5, IncludedFruits: 1, IncludedExtras: 1, Tag: "Pra matar a vontade"},
	{ID: "combo-marmita", Name: "Combo Marmita", Size: "1 litro", PriceCents: 2890, IncludedToppings: 6, IncludedFruits: 2, IncludedExtras: 1, Tag: "Pra compartilhar"},
}

func getMenuCombos(context *gin.Context) {
	context.JSON(200, gin.H{"combos": menuCombos})
}
