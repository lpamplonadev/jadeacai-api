package main

import (
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store, err := openOrderStore(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer store.Close()

	if err := newRouter(store).Run(":" + port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newRouter(store orderStore) *gin.Engine {
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
	router.POST("/api/v1/orders", createOrder(store))

	adminRoutes := router.Group("/api/v1/admin")
	adminRoutes.Use(adminAPIKeyAuth(os.Getenv("ADMIN_API_KEY")))
	adminRoutes.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "ok"})
	})
	adminRoutes.GET("/orders", listOrders(store))

	return router
}

func allowedCORSOrigins() []string {
	configuredOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if configuredOrigins == "" {
		return []string{
			"https://jadeacai-web.vercel.app",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	origins := make([]string, 0)
	for _, origin := range strings.Split(configuredOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{
			"https://jadeacai-web.vercel.app",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	return origins
}
