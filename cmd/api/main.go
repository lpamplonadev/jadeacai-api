package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := newRouter().Run(":" + port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/menu/combos", getMenuCombos)
	router.POST("/api/v1/orders", createOrder)

	return router
}
