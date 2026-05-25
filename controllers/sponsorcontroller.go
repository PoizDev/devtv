package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//'Redis ile Cache'lenme yapılacak ama şimdi değil

func CreateSponsor(c *gin.Context) {
	var sponsors models.Sponsors
	if err := c.BindJSON(&sponsors); err != nil {
		config.Log.Error("Json'ı eşlerken hata oluştu", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	result := in.DB.Create(&sponsors)
	if result.Error != nil {
		config.Log.Error("Sponsor oluşturulurken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sponsor"})
		return
	}
	config.Log.Info("Sponsor oluşturuldu", zap.String("name", sponsors.SponsorName))
	c.JSON(http.StatusOK, gin.H{"message": "Sponsor created successfully"})
}

func GetSponsors(c *gin.Context) {
	var sponsors []models.Sponsors
	result := in.DB.WithContext(c.Request.Context()).Find(&sponsors)
	if result.Error != nil {
		config.Log.Error("Sponsorlar alınırken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sponsors"})
		return
	}
	c.JSON(http.StatusOK, sponsors)
	config.Log.Info("Tüm sponsorlar alındı")
}

func DeleteSponsors(c *gin.Context) {
	sponsorID := c.Param("id")

	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sponsor ID gereklidir"})
		config.Log.Warn("Sponsor ID alanı boş")
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.First(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sponsor bulunamadı"})
		config.Log.Warn("Silinmek istenen sponsor bulunamadı", zap.String("id", sponsorID))
		return
	}

	result := in.DB.Delete(&sponsor)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sponsor silinirken bir hata oluştu"})
		config.Log.Error("Sponsor silme hatası", zap.Error(result.Error))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sponsor başarıyla silindi", "sponsor_id": sponsorID})
}
func UpdateSponsor(c *gin.Context) {
	sponsorID := c.Param("id")

	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sponsor ID gereklidir",
		})
		config.Log.Warn("Sponsor ID alanı boş")
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.First(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sponsor bulunamadı",
		})
		config.Log.Warn("Sponsor bulunamadı")
		return
	}
	if err := c.BindJSON(&sponsor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Geçersiz istek verisi",
		})
		config.Log.Error("Geçersiz istek verisi", zap.Error(err))
		return
	}
	result := in.DB.Save(&sponsor)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Sponsor'u güncellerken bir hata oluştu, " + result.Error.Error(),
		})
		config.Log.Error("Sponsor'u güncellerken bir hata oluştu", zap.Error(result.Error))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Sponsor başarıyla güncellendi",
		"sponsor_id": sponsorID,
	})
}
