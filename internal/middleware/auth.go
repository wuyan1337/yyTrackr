package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

const CurrentUserIDKey = "current_user_id"

// AuthMiddleware creates middleware that requires authentication
func AuthMiddleware(userService *service.UserService, sessionService *service.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if isPublicRoute(path) {
			c.Next()
			return
		}

		hasUsers := userService.HasUsers()

		if !hasUsers {
			if isHTMLRequest(c.Request) {
				c.Redirect(http.StatusFound, "/register")
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Registration required"})
			}
			c.Abort()
			return
		}

		if !sessionService.IsAuthenticated(c.Request) {
			if isHTMLRequest(c.Request) {
				c.Redirect(http.StatusFound, "/login?redirect="+url.QueryEscape(c.Request.URL.Path))
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			}
			c.Abort()
			return
		}

		userID, ok := sessionService.GetCurrentUserID(c.Request)
		if !ok || userID == 0 {
			_ = sessionService.DestroySession(c.Writer, c.Request)
			if isHTMLRequest(c.Request) {
				c.Redirect(http.StatusFound, "/login")
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			}
			c.Abort()
			return
		}

		c.Set(CurrentUserIDKey, userID)
		c.Next()
	}
}

// isPublicRoute checks if a route should be accessible without authentication
func isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/register",
		"/login",
		"/api/auth/register",
		"/api/auth/login",
		"/api/auth/logout",
		"/static/",
		"/favicon.ico",
		"/manifest.json",
		"/healthz",
		"/ical/",
	}

	// API v1 routes use API keys, not session auth
	if strings.HasPrefix(path, "/api/v1/") {
		return true
	}

	for _, route := range publicRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}

func CurrentUserID(c *gin.Context) uint {
	value, exists := c.Get(CurrentUserIDKey)
	if !exists {
		return 0
	}

	if userID, ok := value.(uint); ok {
		return userID
	}

	return 0
}

// isHTMLRequest checks if the request is for HTML content
func isHTMLRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || accept == ""
}

// APIKeyAuth creates middleware that requires API key authentication
func APIKeyAuth(settingsService *service.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		// Also check Authorization: Bearer header
		if apiKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Validate API key
		keyRecord, err := settingsService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Set(CurrentUserIDKey, keyRecord.UserID)
		c.Next()
	}
}
