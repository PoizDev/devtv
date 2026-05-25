package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CreateFacilitator(c *gin.Context) {
	var facilitator models.Facilitators
	if err := c.BindJSON(&facilitator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	result := in.DB.Create(&facilitator)
	if result.Error != nil {
		config.Log.Error("Facilitator oluşturulurken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create facilitator"})
		return
	}
	config.Log.Info("Facilitator oluşturuldu", zap.String("name", facilitator.Name))
	c.JSON(http.StatusOK, gin.H{"message": "Facilitator created successfully"})
}

func GetAllFacilitators(c *gin.Context) {
	var facilitators []models.Facilitators
	result := in.DB.WithContext(c.Request.Context()).Find(&facilitators)
	if result.Error != nil {
		config.Log.Error("Facilitator'lar alınırken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve facilitators"})
		return
	}
	c.JSON(http.StatusOK, facilitators)
	config.Log.Info("Tüm facilitator'lar alındı")
}

func GetFacilitatorsByTopic(c *gin.Context) {
	topic := c.Param("topic")
	var facilitators []models.Facilitators
	result := in.DB.Where("topic = ?", topic).Find(&facilitators)
	if result.Error != nil {
		config.Log.Error("konuya göre konuşmacıları çekerken bir hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve facilitators by topic"})
		return
	}
	config.Log.Debug("Konuya göre konuşmacılar çekildi")
	c.JSON(http.StatusOK, facilitators)
}

func DeleteFacilitator(c *gin.Context) {
	facilID := c.Param("id")

	if facilID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator ID gerekli",
		})
		config.Log.Warn("Facilitator ID boş")
		return
	}

	var facil models.Facilitators
	if err := in.DB.First(&facil, "facilitator_id = ?", facilID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Facilitator Bulunamadı",
		})
		config.Log.Warn("Silinmek istenen facilitator bulunamadı")
		return
	}

	result := in.DB.Delete(&facil)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Facilitator silinirken bir hata oluştu" + result.Error.Error(),
		})
		config.Log.Error("Facilitator silinirken bir hata oluştu", zap.Error(result.Error))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Facilitator başarıyla silindi",
		"facilitator_id": facilID,
	})
	config.Log.Info("Facilitator başarıyla silindi", zap.String("id", facilID))

}

func UpdateFacilitator(c *gin.Context) {
	facilID := c.Param("id")

	if facilID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator ID zorunlu",
		})
		config.Log.Warn("Facilitator ID Boş")
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
		config.Log.Warn("json parse hatası", zap.Error(err))
		return
	}

	var facilitator models.Facilitators
	if err := in.DB.First(&facilitator, facilID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Facilitator bulunamadı",
		})
		config.Log.Warn("Güncellenecek facilitator bulunamadı", zap.String("id", facilID))
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
		config.Log.Error("Facilitator güncellenirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Facilitator güncellenemedi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "Facilitator başarıyla güncellendi",
		"facilitator": facilitator,
	})
	config.Log.Info("Facilitator güncellendi", zap.String("id", facilID), zap.String("name", facilitator.Name))
}
