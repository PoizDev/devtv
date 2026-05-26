package controllers

import (
	"context"
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func invalidateActiveQuestionsCache() {
	if in.RDB != nil {
		ctx := context.Background()
		in.RDB.Del(ctx, activeQuestionsCacheKey)
	}
}

// ---- CATEGORY ----
func GetAllCategories(c *gin.Context) {
	var cats []models.Category
	if err := in.DB.Find(&cats).Error; err != nil {
		config.Log.Error("Kategoriler alınamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kategoriler alınamadı: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func CreateCategory(c *gin.Context) {
	var cat models.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Create(&cat).Error; err != nil {
		config.Log.Error("Kategori oluşturulamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kategori oluşturulamadı"})
		return
	}
	c.JSON(http.StatusOK, cat)
}

func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := in.DB.First(&cat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kategori bulunamadı"})
		return
	}
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Save(&cat).Error; err != nil {
		config.Log.Error("Kategori güncellenemedi", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kategori güncellenemedi"})
		return
	}
	c.JSON(http.StatusOK, cat)
}

func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if err := in.DB.Delete(&models.Category{}, id).Error; err != nil {
		config.Log.Error("Kategori silinemedi", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kategori silinemedi"})
		return
	}
	config.Log.Info("Kategori silindi", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"message": "Kategori silindi"})
}

// ---- TAG ----
func GetAllTags(c *gin.Context) {
	var tags []models.Tag
	if err := in.DB.Preload("Categories").Find(&tags).Error; err != nil {
		config.Log.Error("Tagler alınamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tagler alınamadı: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, tags)
}

func CreateTag(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		CategoryIDs []uint `json:"category_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}

	tag := models.Tag{Name: req.Name}
	if err := in.DB.Create(&tag).Error; err != nil {
		config.Log.Error("Tag oluşturulamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tag oluşturulamadı: " + err.Error()})
		return
	}

	if len(req.CategoryIDs) > 0 {
		var categories []models.Category
		for _, id := range req.CategoryIDs {
			categories = append(categories, models.Category{ID: id})
		}
		if err := in.DB.Model(&tag).Association("Categories").Append(&categories); err != nil {
			config.Log.Error("Tag kategorileri eklenemedi", zap.Error(err))
		}
	}

	in.DB.Preload("Categories").First(&tag, tag.ID)
	c.JSON(http.StatusOK, tag)
}

func UpdateTag(c *gin.Context) {
	id := c.Param("id")
	var tag models.Tag
	if err := in.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag bulunamadı"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		CategoryIDs []uint `json:"category_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}

	err := in.DB.Transaction(func(tx *gorm.DB) error {
		if req.Name != "" {
			tag.Name = req.Name
		}
		if err := tx.Save(&tag).Error; err != nil {
			return err
		}

		if req.CategoryIDs != nil {
			var categories []models.Category
			for _, catID := range req.CategoryIDs {
				categories = append(categories, models.Category{ID: catID})
			}
			if err := tx.Model(&tag).Association("Categories").Replace(categories); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		config.Log.Error("Tag güncellenemedi", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tag güncellenemedi: " + err.Error()})
		return
	}

	in.DB.Preload("Categories").First(&tag, id)
	c.JSON(http.StatusOK, tag)
}

func DeleteTag(c *gin.Context) {
	id := c.Param("id")
	if err := in.DB.Delete(&models.Tag{}, id).Error; err != nil {
		config.Log.Error("Tag silinemedi", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tag silinemedi"})
		return
	}
	config.Log.Info("Tag silindi", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"message": "Tag silindi"})
}

// ---- SURVEY QUESTION ----
func CreateSurveyQuestion(c *gin.Context) {
	var sq models.SurveyQuestion
	if err := c.ShouldBindJSON(&sq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Create(&sq).Error; err != nil {
		config.Log.Error("Soru oluşturulamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Soru oluşturulamadı"})
		return
	}
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, sq)
}

func UpdateSurveyQuestion(c *gin.Context) {
	id := c.Param("id")
	var sq models.SurveyQuestion
	if err := in.DB.First(&sq, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Soru bulunamadı"})
		return
	}
	if err := c.ShouldBindJSON(&sq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Save(&sq).Error; err != nil {
		config.Log.Error("Soru güncellenemedi", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Soru güncellenemedi"})
		return
	}
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, sq)
}

func DeleteSurveyQuestion(c *gin.Context) {
	id := c.Param("id")
	if err := in.DB.Delete(&models.SurveyQuestion{}, id).Error; err != nil {
		config.Log.Error("Soru silinemedi", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Soru silinemedi"})
		return
	}
	config.Log.Info("Soru silindi", zap.String("id", id))
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, gin.H{"message": "Soru silindi"})
}

// ---- SURVEY OPTION ----
func CreateSurveyOption(c *gin.Context) {
	var so models.SurveyOption
	if err := c.ShouldBindJSON(&so); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Create(&so).Error; err != nil {
		config.Log.Error("Şık oluşturulamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Şık oluşturulamadı"})
		return
	}
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, so)
}

func UpdateSurveyOption(c *gin.Context) {
	id := c.Param("id")
	var so models.SurveyOption
	if err := in.DB.First(&so, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Şık bulunamadı"})
		return
	}
	if err := c.ShouldBindJSON(&so); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek"})
		return
	}
	if err := in.DB.Save(&so).Error; err != nil {
		config.Log.Error("Şık güncellenemedi", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Şık güncellenemedi"})
		return
	}
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, so)
}

func DeleteSurveyOption(c *gin.Context) {
	id := c.Param("id")
	if err := in.DB.Delete(&models.SurveyOption{}, id).Error; err != nil {
		config.Log.Error("Şık silinemedi", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Şık silinemedi"})
		return
	}
	config.Log.Info("Şık silindi", zap.String("id", id))
	invalidateActiveQuestionsCache()
	c.JSON(http.StatusOK, gin.H{"message": "Şık silindi"})
}

// ---- ADMIN: GET ALL QUESTIONS (ACTIVE & INACTIVE) ----
func GetAllSurveyQuestions(c *gin.Context) {
	var questions []models.SurveyQuestion
	// Preload Options and Option's Tag for admin view
	err := in.DB.Preload("Options.Tag.Categories").Order("\"order\" asc").Find(&questions).Error
	if err != nil {
		config.Log.Error("Tüm sorular alınamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sorular alınırken bir hata oluştu: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, questions)
}
