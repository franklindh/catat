package api

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		ctx.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		ctx.Header("X-Content-Type-Options", "nosniff")

		ctx.Header("X-Frame-Options", "DENY")

		ctx.Header("X-XSS-Protection", "1; mode=block")

		ctx.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		ctx.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		ctx.Header("Pragma", "no-cache")
		ctx.Header("Expires", "0")

		ctx.Next()
	}
}

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

type RateLimiter struct {
	clients sync.Map
	rate    int
	window  time.Duration
}

type clientData struct {
	count     int
	startTime time.Time
	mu        sync.Mutex
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:   rate,
		window: window,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		rl.clients.Range(func(key, value interface{}) bool {
			client := value.(*clientData)
			client.mu.Lock()
			if now.Sub(client.startTime) > rl.window {
				rl.clients.Delete(key)
			}
			client.mu.Unlock()
			return true
		})
	}
}

func getClientIP(ctx *gin.Context) string {

	if xff := ctx.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xri := ctx.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	return ctx.ClientIP()
}

func (rl *RateLimiter) Allow(clientIP string) (bool, int, time.Time) {
	now := time.Now()

	value, _ := rl.clients.LoadOrStore(clientIP, &clientData{
		count:     0,
		startTime: now,
	})
	client := value.(*clientData)

	client.mu.Lock()
	defer client.mu.Unlock()

	if now.Sub(client.startTime) > rl.window {
		client.count = 0
		client.startTime = now
	}

	remaining := rl.rate - client.count
	resetTime := client.startTime.Add(rl.window)

	if client.count >= rl.rate {
		return false, 0, resetTime
	}

	client.count++
	remaining = rl.rate - client.count

	return true, remaining, resetTime
}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := getClientIP(ctx)

		allowed, remaining, resetTime := limiter.Allow(clientIP)

		ctx.Header("X-RateLimit-Limit", strconv.Itoa(limiter.rate))
		ctx.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		ctx.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			retryAfter := max(int(time.Until(resetTime).Seconds()), 1)
			ctx.Header("Retry-After", strconv.Itoa(retryAfter))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		ctx.Next()
	}
}
