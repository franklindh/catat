package api

import (
	"net/http"
	"strings"

	"github.com/franklindh/catat/token"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey        = "Authorization"
	authorizationHeaderBearerType = "bearer"
	authorizationPayloadKey       = "authorization_payload"

	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

func authMiddleware(maker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader(authorizationHeaderKey)
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No header was passed"})
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) != 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or Missing Bearer Token"})
			return
		}

		authType := strings.ToLower(fields[0])
		if authType != authorizationHeaderBearerType {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization Type Not Supported"})
			return
		}

		token := fields[1]
		payload, err := maker.VerifyToken(token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Access Token Not Valid"})
			return
		}

		ctx.Set(authorizationPayloadKey, payload)

		ctx.Next()
	}
}

func requireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, exists := ctx.Get(authorizationPayloadKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		authPayload, ok := payload.(*token.Payload)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization payload"})
			return
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(authPayload.Role, role) {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied: insufficient role"})
	}
}
