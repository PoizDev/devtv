package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateFacilitator(c *gin.Context) {
	type CreateFacilitatorReq struct {
		Name         string `json:"name" binding:"required"`
		Title        string `json:"title" binding:"required"`
		Topic        string `json:"topic" binding:"required"`
		TagIDs       []uint `json:"tag_ids"`
		TopicDetails string `json:"topic_details" binding:"required"`
		Photograph   string `json:"photograph" binding:"required"`
	}

	var req CreateFacilitatorReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var tags []models.Tag
	if len(req.TagIDs) > 0 {
		in.DB.Find(&tags, req.TagIDs)
	}

	facilitator := models.Facilitators{
		Name:         req.Name,
		Title:        req.Title,
		Topic:        req.Topic,
		Tags:         tags,
		TopicDetails: req.TopicDetails,
		Photograph:   req.Photograph,
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
	result := in.DB.WithContext(c.Request.Context()).Preload("Tags").Find(&facilitators)
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
	result := in.DB.Where("topic = ?", topic).Preload("Tags").Find(&facilitators)
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
		config.Log.Error("Facilitator silinirken hata", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Facilitator silinemedi"})
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
		Name         string `json:"name"`
		Title        string `json:"title"`
		Topic        string `json:"topic"`
		TagIDs       []uint `json:"tag_ids"`
		TopicDetails string `json:"topic_details"`
		Photograph   string `json:"photograph"` //path/to/photograph
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
	err := in.DB.Transaction(func(tx *gorm.DB) error {
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

		if len(updateData) > 0 {
			if err := tx.Model(&facilitator).Updates(updateData).Error; err != nil {
				return err
			}
		}

		if req.TagIDs != nil {
			var tags []models.Tag
			if len(req.TagIDs) > 0 {
				if err := tx.Find(&tags, req.TagIDs).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&facilitator).Association("Tags").Replace(&tags); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		config.Log.Error("Facilitator güncellenirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Facilitator güncellenemedi"})
		return
	}

	in.DB.Preload("Tags").First(&facilitator, facilID)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Facilitator başarıyla güncellendi",
		"facilitator": facilitator,
	})
	config.Log.Info("Facilitator güncellendi", zap.String("id", facilID), zap.String("name", facilitator.Name))
}
