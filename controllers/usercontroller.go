package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// İzin verilen roller — signup'ta sadece bunlar kabul edilir
var allowedRoles = map[string]bool{
	"user":      true,
	"moderator": true,
	// "admin" burada YOK — admin rolü sadece mevcut admin tarafından atanabilir
}

func Signup(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		config.Log.Error("Json'ı eşlerken hata oluştu", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username ve password zorunludur"})
		return
	}

	// Şifre uzunluk kontrolü
	if len(body.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Şifre en az 6 karakter olmalıdır"})
		return
	}

	// Role validasyonu — boşsa "user" yap, izin verilmeyen rol gelirse reddet
	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role == "" {
		role = "user"
	}
	if !allowedRoles[role] {
		config.Log.Warn("Yetkisiz rol denemesi", zap.String("requested_role", body.Role), zap.String("username", body.Username))
		c.JSON(http.StatusForbidden, gin.H{"error": "Bu rol ile kayıt olunamazsınız"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		config.Log.Error("Şifre hashlenirken bir hata oluştu", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username: body.Username,
		Password: string(hash),
		Role:     role,
	}
	result := in.DB.Create(&user)
	if result.Error != nil {
		config.Log.Error("Kullanıcı oluşturulurken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	config.Log.Info("Kullanıcı başarıyla oluşturuldu", zap.String("username", user.Username), zap.String("role", user.Role))
	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

func Login(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		config.Log.Error("Json'ı eşlerken hata oluştu", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username ve password zorunludur"})
		return
	}

	var user models.User
	result := in.DB.Where("username = ?", body.Username).First(&user)
	if result.Error != nil {
		config.Log.Error("Kullanıcı bulunamadı", zap.Error(result.Error))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		config.Log.Error("Şifre doğrulanamadı", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// JWT Secret kontrolü
	jwtSecret := in.Auth.JWTSecret
	if jwtSecret == "" {
		config.Log.Error("JWT_SECRET tanımlanmamış! Login yapılamaz.")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sunucu yapılandırma hatası"})
		return
	}

	// Token süresi config'den alınıyor
	expiryDays := in.Auth.TokenExpiryDays
	if expiryDays <= 0 {
		expiryDays = 30 // Varsayılan
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.UserID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24 * time.Duration(expiryDays)).Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		config.Log.Error("Token imzalanırken hata oluştu", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Cookie ayarları config'den okunuyor
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"Auth",
		tokenString,
		3600*24*expiryDays,
		"/",
		in.Auth.CookieDomain,
		in.Auth.CookieSecure,
		true, // HttpOnly — JavaScript erişemez
	)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
	config.Log.Info("Kullanıcı giriş yaptı", zap.String("username", user.Username))
}

func GetAllUsers(c *gin.Context) {
	var users []models.User
	result := in.DB.Select("user_id", "username", "role", "created_at").Find(&users)
	if result.Error != nil {
		config.Log.Error("Kullanıcılar alınırken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	c.JSON(http.StatusOK, users)
	config.Log.Info("Tüm kullanıcılar alındı", zap.Uint("userID", c.GetUint("userID")))
}

func DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID gerekli",
		})
		config.Log.Warn("User ID parametresi boş")
		return
	}

	var user models.User
	if err := in.DB.First(&user, "user_id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Silinmek istenen user bulunamadı",
		})
		config.Log.Warn("Silinmek istenen user bulunamadı", zap.String("id", userID))
		return
	}

	result := in.DB.Delete(&user)
	if result.Error != nil {
		config.Log.Error("User silinirken hata", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User silinemedi, " + result.Error.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "User silindi",
		"user_id": userID,
	})
	config.Log.Info("User silindi", zap.String("id", userID))
}

func UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID Gerekli",
		})
		config.Log.Warn("User ID parametresi boş")
		return
	}
	var body struct {
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.BindJSON(&body); err != nil {
		config.Log.Error("Json'ı eşlerken hata oluştu", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	var user models.User
	if err := in.DB.First(&user, "user_id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Güncellenmek istenen user bulunamadı",
		})
		config.Log.Warn("Güncellenmek istenen user bulunamadı", zap.String("id", userID))
		return
	}
	if body.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
		if err != nil {
			config.Log.Error("Şifre hashlenirken hata oluştu", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Şifre hashlenirken hata oluştu"})
			return
		}
		user.Password = string(hash)
	}
	if body.Role != "" {
		user.Role = body.Role
	}
	result := in.DB.Save(&user)
	if result.Error != nil {
		config.Log.Error("User güncellenirken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User güncellenirken bir hata oluştu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User güncellendi"})
	config.Log.Info("User güncellendi", zap.String("id", userID))
}
