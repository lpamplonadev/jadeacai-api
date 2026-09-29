package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
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
	router.Use(gin.Logger(), gin.Recovery(), cors.New(cors.Config{
		AllowOrigins: allowedCORSOrigins(),
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/menu/combos", getMenuCombos)
	router.POST("/api/v1/orders", createOrder)

	return router
}

func allowedCORSOrigins() []string {
	configuredOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if configuredOrigins == "" {
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	origins := make([]string, 0)
	for _, origin := range strings.Split(configuredOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	return origins
}
