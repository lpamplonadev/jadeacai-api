package main

import (
	"net/http"

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

func createOrder(context *gin.Context) {
	var request createOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid order payload"})
		return
	}

	context.JSON(http.StatusAccepted, gin.H{
		"status":    "received",
		"persisted": false,
	})
}
