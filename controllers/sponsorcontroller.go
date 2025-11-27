package controllers

import (
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
)

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
