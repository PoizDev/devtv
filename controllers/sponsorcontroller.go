package controllers

import (
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
)

//'Redis ile Cache'lenme yapılacak ama şimdi değil

func CreateSponsor(c *gin.Context) {
	var sponsors models.Sponsors
	if err := c.BindJSON(&sponsors); err != nil {
		log.Error("Json'ı eşlerken hata oluştu: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	result := in.DB.Create(&sponsors)
	if result.Error != nil {
		log.Error("Sponsor oluşturulurken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sponsor"})
		return
	}
	log.Info("Sponsor oluşturuldu: ", sponsors.SponsorName)
	c.JSON(http.StatusOK, gin.H{"message": "Sponsor created successfully"})
}

func GetSponsors(c *gin.Context) {
	var sponsors []models.Sponsors
	result := in.DB.Find(&sponsors)
	if result.Error != nil {
		log.Error("Sponsorlar alınırken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sponsors"})
		return
	}
	c.JSON(http.StatusOK, sponsors)
	log.Info("Tüm sponsorlar alındı")
}

func DeleteSponsors(c *gin.Context) {
	sponsorID := c.Param("id")

	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sponsor ID gereklidir",
		})
		log.Warn("Sponsor ID alanı boş")
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.Find(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sponsor bulunamadı",
		})
		log.Warn("Sponsor bulunamadı")
		return
	}
	result := in.DB.Delete(&sponsor)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Sponsor'u silerken bir hata oluştu, " + result.Error.Error(),
		})
		log.Error("Sponsor'u silerken biğr hata oluştu, ", result.Error)
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Sponsor başarıyla silindi",
		"sponsor_id": sponsorID,
	})
}

func UpdateSponsor(c *gin.Context) {
	sponsorID := c.Param("id")

	if sponsorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sponsor ID gereklidir",
		})
		log.Warn("Sponsor ID alanı boş")
		return
	}

	var sponsor models.Sponsors
	if err := in.DB.First(&sponsor, "sponsor_id = ?", sponsorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sponsor bulunamadı",
		})
		log.Warn("Sponsor bulunamadı")
		return
	}
	if err := c.BindJSON(&sponsor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Geçersiz istek verisi",
		})
		log.Error("Geçersiz istek verisi: ", err)
		return
	}
	result := in.DB.Save(&sponsor)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Sponsor'u güncellerken bir hata oluştu, " + result.Error.Error(),
		})
		log.Error("Sponsor'u güncellerken bir hata oluştu, ", result.Error)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Sponsor başarıyla güncellendi",
		"sponsor_id": sponsorID,
	})
}
