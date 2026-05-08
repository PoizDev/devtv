package middlewares

import (
	"context"
	"devtv/in"
	"devtv/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/jeanphorn/log4go"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("Auth")
		if err != nil {
			log.Warn("Auth cookie bulunamadı")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied - Token bulunamadı"})
			c.Abort()
			return
		}

		jwtSecret := in.Auth.JWTSecret
		if jwtSecret == "" {
			log.Critical("JWT_SECRET tanımlanmamış! Auth doğrulaması yapılamaz.")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sunucu yapılandırma hatası"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Signing method kontrolü — sadece HMAC kabul et
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("beklenmeyen signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
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

func RequestLoggerMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		//'Health' url'leri için loglamayı atla
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/health/check" {
			c.Next()
			return
		}

		c.Next()

		duration := time.Since(start)
		log.Info(fmt.Sprintf(
			"%s %s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.Writer.Status(),
			duration,
		))
	}
}

func FormatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}

	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
	}

	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}

	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}

	// Eğer hiçbir şey yoksa (< 1 dakika), saniye göster
	if len(parts) == 0 {
		if seconds == 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", seconds)
	}

	// İlk 2 parçayı birleştir (örn: "2 hours 30 minutes")
	if len(parts) > 2 {
		parts = parts[:2]
	}

	if len(parts) > 3 {
		parts = parts[:3]
	}

	return joinParts(parts)
}

// joinParts - String slice'ı birleştir
func joinParts(parts []string) string {
	if len(parts) == 0 {
		return "0 seconds"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + " " + parts[1]
}
