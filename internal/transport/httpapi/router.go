package httpapi

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
)

func NewRouter(service *application.Service, adminAPIKey string) *gin.Engine {
	router := gin.New()
	// Initialize the router with logging, recovery, and CORS middleware
	router.Use(gin.Logger(), gin.Recovery(), cors.New(cors.Config{
		AllowOrigins: allowedCORSOrigins(),
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       12 * time.Hour,
	}))
	// Health check route
	router.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "ok"})
	})
	// Public API routes
	router.GET("/api/v1/store/status", getStoreStatus(service))
	router.GET("/api/v1/menu/catalog", getMenuCatalog(service))
	router.GET("/api/v1/menu/combos", getMenuCombos(service))
	router.GET("/api/v1/orders/:orderId/tracking", trackOrder(service))
	router.POST("/api/v1/orders", createOrder(service))
	// Admin routes
	adminRoutes := router.Group("/api/v1/admin")
	adminRoutes.Use(adminAPIKeyAuth(adminAPIKey))
	adminRoutes.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "ok"})
	})
	// Admin settings routes
	adminRoutes.GET("/settings", getStoreSettings(service))
	adminRoutes.PATCH("/settings", updateStoreSettings(service))
	adminRoutes.PATCH("/settings/override", updateStoreOverride(service))
	// Admin dashboard routes
	adminRoutes.GET("/dashboard", getDashboard(service))
	adminRoutes.GET("/orders", listOrders(service))
	adminRoutes.PATCH("/orders/:orderId", updateOrderStatus(service))
	// Admin catalog routes
	adminRoutes.GET("/catalog", getCatalog(service))
	// Items catalog routes
	adminRoutes.POST("/catalog/items", createCatalogItem(service))
	adminRoutes.PATCH("/catalog/items/:itemId", updateCatalogItem(service))
	adminRoutes.DELETE("/catalog/items/:itemId", archiveCatalogItem(service))
	// Combos catalog routes 
	adminRoutes.POST("/catalog/combos", createCatalogCombo(service))
	adminRoutes.PATCH("/catalog/combos/:comboId", updateCatalogCombo(service))
	adminRoutes.DELETE("/catalog/combos/:comboId", archiveCatalogCombo(service))
	// Gourmet catalog routes 
	adminRoutes.POST("/catalog/gourmets", createCatalogGourmet(service))
	adminRoutes.PATCH("/catalog/gourmets/:gourmetId", updateCatalogGourmet(service))
	adminRoutes.DELETE("/catalog/gourmets/:gourmetId", archiveCatalogGourmet(service))


	return router
}

func allowedCORSOrigins() []string {
	configuredOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if configuredOrigins == "" {
		return defaultCORSOrigins()
	}

	origins := make([]string, 0)
	for _, origin := range strings.Split(configuredOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return defaultCORSOrigins()
	}

	return origins
}

func defaultCORSOrigins() []string {
	return []string{
		"https://jadesacai.vercel.app",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
}
