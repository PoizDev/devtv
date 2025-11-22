package controllers

import (
	"devtv/in"
	"devtv/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/jeanphorn/log4go"
	"golang.org/x/crypto/bcrypt"
)

func Signup(c *gin.Context) {
	log.LoadConfiguration("./log4go.json")
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := c.BindJSON(&body); err != nil {
		log.Error("Json'ı eşlerken hata oluştu: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		log.Error("Şifre hashlenirken bir hata oluştu: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username: body.Username,
		Password: string(hash),
		Role:     body.Role,
	}
	result := in.DB.Create(&user)
	if result.Error != nil {
		log.Error("Kullanacı oluşturulurken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	log.Info("Kullanıcı başarıyla oluşturuldu: ", user.Username)
	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

func Login(c *gin.Context) {
	log.LoadConfiguration("./log4go.json")
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if c.BindJSON(&body) != nil {
		log.Error("Json'ı eşlerken hata oluştu")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	log.Info("Json başarıyla eşlendi")
	var user models.User
	result := in.DB.Where("username = ?", body.Username).First(&user)
	if result.Error != nil {
		log.Error("Kullanıcı bulunamadı: ", result.Error)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		log.Error("Şifre doğrulanamadı: ", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.UserID,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		log.Error("Token imzalanırken hata oluştu: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("Auth", tokenString, 3600*24*30, "/", "localhost", false, true) // Canlıya çıkarken Secure true yapılacak
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
	log.Info("Kullanıcı başarıyla giriş yaptı: ", user.Username)
}

func GetAllUsers(c *gin.Context) {
	log.LoadConfiguration("./log4go.json")
	var users []models.User
	result := in.DB.Find(&users)
	if result.Error != nil {
		log.Error("Kullanıcılar alınırken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	c.JSON(http.StatusOK, users)
	log.Info("Tüm kullanıcılar başarıyla alındı")
}
