package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CreateSponsor(c *gin.Context) {
	var sponsors models.Sponsors
	if err := c.ShouldBindJSON(&sponsors); err != nil {
		config.Log.Error("Json'ı eşlerken hata oluştu", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek verisi"})
		return
	}
	result := in.DB.Create(&sponsors)
	if result.Error != nil {
		config.Log.Error("Sponsor oluşturulurken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sponsor oluşturulamadı"})
		return
	}
	config.Log.Info("Sponsor oluşturuldu", zap.String("name", sponsors.SponsorName))
	c.JSON(http.StatusOK, gin.H{"message": "Sponsor başarıyla oluşturuldu"})
}

func GetSponsors(c *gin.Context) {
	var sponsors []models.Sponsors
	result := in.DB.WithContext(c.Request.Context()).Find(&sponsors)
	if result.Error != nil {
		config.Log.Error("Sponsorlar alınırken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sponsorlar alınamadı"})
		return
	}
	c.JSON(http.StatusOK, sponsors)
}

func DeleteSponsors(c *gin.Context) {
	sponsorID := c.Param("id")
	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sponsor ID gereklidir"})
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.First(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sponsor bulunamadı"})
		return
	}

	if result := in.DB.Delete(&sponsor); result.Error != nil {
		config.Log.Error("Sponsor silme hatası", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sponsor silinemedi"})
		return
	}

	config.Log.Info("Sponsor silindi", zap.String("id", sponsorID))
	c.JSON(http.StatusOK, gin.H{"message": "Sponsor başarıyla silindi", "sponsor_id": sponsorID})
}

func UpdateSponsor(c *gin.Context) {
	sponsorID := c.Param("id")
	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sponsor ID gereklidir"})
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.First(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sponsor bulunamadı"})
		return
	}

	if err := c.ShouldBindJSON(&sponsor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek verisi"})
		return
	}

	if result := in.DB.Save(&sponsor); result.Error != nil {
		config.Log.Error("Sponsor güncellenirken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sponsor güncellenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Sponsor başarıyla güncellendi",
		"sponsor_id": sponsorID,
	})
}
