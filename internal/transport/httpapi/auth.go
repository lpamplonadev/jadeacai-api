package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const minimumAdminAPIKeyLength = 32

func adminAPIKeyAuth(expectedKey string) gin.HandlerFunc {
	return func(context *gin.Context) {
		if len(expectedKey) < minimumAdminAPIKeyLength {
			context.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin API authentication is not configured"})
			return
		}

		scheme, providedKey, hasBearerToken := strings.Cut(context.GetHeader("Authorization"), " ")
		if !hasBearerToken || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare([]byte(providedKey), []byte(expectedKey)) != 1 {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		context.Next()
	}
}
