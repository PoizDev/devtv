package controllers

import (
	"devtv/in"
	"devtv/models"
	"encoding/json"
	"net/http"

	log "github.com/jeanphorn/log4go"

	"github.com/gin-gonic/gin"
)

func CreateFacilitator(c *gin.Context) {
	var facilitator models.Facilitators
	if err := c.BindJSON(&facilitator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	result := in.DB.Create(&facilitator)
	if result.Error != nil {
		log.Error("Facilitator oluşturulurken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create facilitator"})
		return
	}
	log.Info("Facilitator oluşturuldu: ", facilitator.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Facilitator created successfully"})
}

func GetAllFacilitators(c *gin.Context) {
	var facilitators []models.Facilitators
	result := in.DB.WithContext(c.Request.Context()).Find(&facilitators)
	if result.Error != nil {
		log.Error("Facilitator'lar alınırken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve facilitators"})
		return
	}
	c.JSON(http.StatusOK, facilitators)
	log.Info("Tüm facilitator'lar alındı")
}

func GetFacilitatorsByTopic(c *gin.Context) {
	topic := c.Param("topic")
	var facilitators []models.Facilitators
	result := in.DB.Where("topic = ?", topic).Find(&facilitators)
	if result.Error != nil {
		log.Error("konuya göre konuşmacıları çekerken bir hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve facilitators by topic"})
		return
	}
	log.Fine("Konuya göre konuşmacılar çekildi")
	c.JSON(http.StatusOK, facilitators)
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

	var facil models.Facilitators
	if err := in.DB.First(&facil, "facilitator_id = ?", facilID).Error; err != nil {
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
		log.Error("Facilitator silinirken bir hata oluştu: ", result.Error)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Facilitator başarıyla silindi",
		"facilitator_id": facilID,
	})
	log.Info("Facilitator başarıyla silindi - ID", facilID)

}

func UpdateFacilitator(c *gin.Context) {
	facilID := c.Param("id")

	if facilID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator ID zorunlu",
		})
		log.Warn("Facilitator ID Boş")
		return
	}

	type UpdateFacilitatorReq struct {
		Name         string   `json:"name"`
		Title        string   `json:"title"`
		Topic        string   `json:"topic"`
		Tags         []string `json:"tags"`
		TopicDetails string   `json:"topic_details"`
		Photograph   string   `json:"photograph"` //path/to/photograph
	}
	var req UpdateFacilitatorReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON Parse ederken hata oluştu " + err.Error()})
		log.Warn("json parse hatası: ", err)
		return
	}

	var facilitator models.Facilitators
	if err := in.DB.First(&facilitator, facilID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator bulunamadı",
		})
		log.Warn("Güncellenecek facilitator bulunamadı - ID ", facilID)
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
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		updateData["tags"] = tagsJSON
	}
	if req.TopicDetails != "" {
		updateData["topic_details"] = req.TopicDetails
	}
	if req.Photograph != "" {
		updateData["photograph"] = req.Photograph
	}

	if err := in.DB.Model(&facilitator).Updates(updateData).Error; err != nil {
		log.Error("Facilitator güncellenirken hata: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Facilitator güncellenemedi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "Facilitator başarıyla güncellendi",
		"facilitator": facilitator,
	})
	log.Info("Facilitator güncellendi - ID: ", facilID, "İsim: ", facilitator.Name)
}
