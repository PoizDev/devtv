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

func DeleteFacilitator(c *gin.Context) {
	facilID := c.Param("id")

	if facilID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator ID gerekli",
		})
		log.Warn("Facilitator ID boş")
		return
	}

	var facil models.Faciliators
	if err := in.DB.First(&facil, "faciliator_id = ?", facilID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Facilitator Bulunamadı",
		})
		log.Warn("Silinmek istenen facilitator bulunamadı")
		return
	}

	result := in.DB.Delete(&facil)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Facilitator silinirken bir hata oluştu" + result.Error.Error(),
		})
		log.Error("Facilitator oluşturulurken bir hata oluştu: ", result.Error)
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Facilitator başarıyla silindi",
		"facilitator_id": facilID,
	})
	log.Info("Facilitator başarıyla silindi - ID", facilID)
}

func UpdateFaciliator(c *gin.Context) {
	facilID := c.Param("id")

	if facilID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Faciliator ID zorunlu",
		})
		log.Warn("Facilitator ID Boş")
	}

	type UpdateFacilitatorReq struct {
		Name         string `json:"name"`
		Title        string `json:"title"`
		Topic        string `json:"topic"`
		TopicDetails string `json:"topic_details"`
		Photograph   string `json:"photograph"` //path/to/photograoh
	}
	var req UpdateFacilitatorReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON Parse ederken hata oluştu " + err.Error()})
		log.Warn("json parse hatası: ", err)
		return
	}

	var faciliator models.Faciliators
	if err := in.DB.First(&faciliator, facilID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Workshop bulunamadı",
		})
		log.Warn("Güncellenecek workshop bulunamadı- ID ", facilID)
		return
	}
	updateData := map[string]interface{}{}

	if req.Name != "" {
		updateData["name"] = req.Name
	}
	if req.Title != "" {
		updateData["title"] = req.Title
	}
	if req.Topic != "" {
		updateData["topic"] = req.Topic
	}
	if req.TopicDetails != "" {
		updateData["topic_details"] = req.TopicDetails
	}
	if req.Photograph != "" {
		updateData["photograph"] = req.Photograph
	}

	if err := in.DB.Model(&faciliator).Updates(updateData).Error; err != nil {
		log.Error("Faciliator güncellenirken hata: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Faciliator güncellenemedi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Faciliator başarıyla güncellendi",
		"faciliator": faciliator,
	})
	log.Info("Faciliator güncellendi - ID: ", facilID, "İsim: ", faciliator.Name)
}
