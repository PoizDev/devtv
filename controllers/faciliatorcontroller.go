package controllers

import (
	"devtv/in"
	"devtv/models"
	"net/http"

	log "github.com/jeanphorn/log4go"

	"github.com/gin-gonic/gin"
)

func CreateFaciliator(c *gin.Context) {
	var faciliator models.Faciliators
	if err := c.BindJSON(&faciliator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	result := in.DB.Create(&faciliator)
	if result.Error != nil {
		log.Error("Faciliator oluşturulurken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create faciliator"})
		return
	}
	log.Info("Faciliator oluşturuldu: ", faciliator.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Faciliator created successfully"})
}

func GetAllFaciliators(c *gin.Context) {
	var faciliators []models.Faciliators
	result := in.DB.Find(&faciliators)
	if result.Error != nil {
		log.Error("Faciliatorlar alınırken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve faciliators"})
		return
	}
	c.JSON(http.StatusOK, faciliators)
	log.Info("Tüm faciliatorlar alındı")
}

func GetFaciliatorsByTopic(c *gin.Context) {
	topic := c.Param("topic")
	var faciliators []models.Faciliators
	result := in.DB.Where("topic = ?", topic).Find(&faciliators)
	if result.Error != nil {
		log.Error("konuya göre konuşmacıları çekerken bir hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve faciliators by topic"})
		return
	}
	log.Fine("Konuya göre konuşmacılar çekildi")
	c.JSON(http.StatusOK, faciliators)
}
