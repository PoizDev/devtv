package middlewares

import (
	"context"
	"devtv/in"
	"devtv/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/jeanphorn/log4go"
)

// AuthMiddleware - Token kontrol et
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("Auth")
		if err != nil {
			log.Warn("Auth cookie bulunamadı")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied - Token bulunamadı"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err != nil || !token.Valid {
			log.Warn("Token doğrulanamadı: ", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied - Geçersiz token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := uint(claims["sub"].(float64))

		var user models.User
		if err := in.DB.First(&user, userID).Error; err != nil {
			log.Error("Kullanıcı bulunamadı: ", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kullanıcı bulunamadı"})
			c.Abort()
			return
		}

		// Sadece admin erişebilir
		if strings.ToLower(user.Role) != "admin" {
			log.Warn("Yetkisiz erişim denemesi - Kullanıcı: ", user.Username, " Role: ", user.Role)
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu işlemi yapma yetkiniz yok - Sadece admin erişebilir"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("user", user)
		c.Next()
	}
}

func TimeoutMiddleware(timeout time.Duration) func(*gin.Context) {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded {
			log.Warn("İstek zaman aşımına uğradı: ", c.Request.URL.Path)
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "İstek zaman aşımına uğradı"})
			c.Abort()
		}
	}
}
