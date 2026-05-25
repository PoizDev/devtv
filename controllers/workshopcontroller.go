package controllers

import (
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateWorkshopWithSlots(c *gin.Context) {
	type CreateWorkshopRequest struct {
		WorkshopName string                    `json:"workshop_name" binding:"required"`
		WorkshopDate time.Time                 `json:"workshop_date" binding:"required"`
		TimeSlots    []models.WorkshopTimeSlot `json:"time_slots"`
	}

	var req CreateWorkshopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Slot'lar varsa order ekle
	if len(req.TimeSlots) > 0 {
		for i := range req.TimeSlots {
			req.TimeSlots[i].SlotOrder = i + 1
		}
	}

	workshop := models.Workshops{
		WorkshopName: req.WorkshopName,
		WorkshopDate: req.WorkshopDate,
		TimeSlots:    req.TimeSlots,
	}

	err := in.DB.Create(&workshop).Error
	if err != nil {
		config.Log.Error("Workshop oluşturulurken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Workshop oluşturulamadı"})
		return
	}

	message := fmt.Sprintf("Workshop oluşturuldu: %s", workshop.WorkshopName)
	if len(req.TimeSlots) > 0 {
		message += fmt.Sprintf(" (%d slot eklendi)", len(req.TimeSlots))
	} else {
		message += " (slot'sız)"
	}

	config.Log.Info(message)
	c.JSON(http.StatusCreated, gin.H{
		"message":  message,
		"workshop": workshop,
	})
}

func AddSlotsToWorkshop(c *gin.Context) {
	workshopID := c.Param("id")

	type AddSlotsRequest struct {
		TimeSlots []models.WorkshopTimeSlot `json:"time_slots" binding:"required"`
	}

	var req AddSlotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var workshop models.Workshops
	if err := in.DB.First(&workshop, workshopID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		return
	}

	for _, slot := range req.TimeSlots {
		var facilitator models.Facilitators
		if err := in.DB.First(&facilitator, slot.FacilitatorID).Error; err != nil {
			config.Log.Error("Facilitator bulunamadı", zap.Uint("facilitatorID", slot.FacilitatorID))
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Facilitator ID'si %d olan kişi bulunamadı", slot.FacilitatorID)})
			return
		}
	}

	var maxOrder int
	in.DB.Model(&models.WorkshopTimeSlot{}).
		Where("workshop_id = ?", workshopID).
		Select("COALESCE(MAX(slot_order), 0)").
		Scan(&maxOrder)

	for i := range req.TimeSlots {
		req.TimeSlots[i].WorkshopID = workshop.WorkshopID
		req.TimeSlots[i].SlotOrder = maxOrder + i + 1
	}

	if err := in.DB.Create(&req.TimeSlots).Error; err != nil {
		config.Log.Error("Slot'lar eklenirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot'lar eklenemedi"})
		return
	}

	config.Log.Info("Workshop'a yeni slotlar eklendi", zap.String("workshopID", workshopID), zap.Int("slotsCount", len(req.TimeSlots)))
	c.JSON(http.StatusCreated, gin.H{
		"message":     "Slot'lar başarıyla eklendi",
		"added_slots": len(req.TimeSlots),
		"slots":       req.TimeSlots,
	})
}

func GetWorkshopSchedule(c *gin.Context) {
	workshopID := c.Param("id")

	var workshop models.Workshops
	err := in.DB.WithContext(c.Request.Context()).
		Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
			return db.Order("slot_order ASC")
		}).
		Preload("TimeSlots.Facilitator").
		First(&workshop, workshopID).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		return
	}

	now := time.Now()
	var currentSlot *models.TimeSlotResponse
	var allSlots []models.TimeSlotResponse

	for _, slot := range workshop.TimeSlots {
		slotResponse := models.TimeSlotResponse{
			SlotID:    slot.SlotID,
			SlotStart: slot.SlotStart,
			SlotEnd:   slot.SlotEnd,
			SlotOrder: slot.SlotOrder,
			Facilitator: models.FacilitatorResponse{
				FacilitatorID: slot.Facilitator.FacilitatorID,
				Name:          slot.Facilitator.Name,
				Topic:         slot.Facilitator.Topic,
				Tags:          slot.Facilitator.Tags,
				TopicDetails:  slot.Facilitator.TopicDetails,
				Photograph:    slot.Facilitator.Photograph,
			},
		}

		if now.After(slot.SlotStart) && now.Before(slot.SlotEnd) {
			currentSlot = &slotResponse
		}

		allSlots = append(allSlots, slotResponse)
	}

	response := models.WorkshopScheduleResponse{
		WorkshopID:   workshop.WorkshopID,
		WorkshopName: workshop.WorkshopName,
		WorkshopDate: workshop.WorkshopDate,
		CurrentSlot:  currentSlot,
		AllSlots:     allSlots,
		TotalSlots:   len(allSlots),
	}

	c.JSON(http.StatusOK, response)
}

func GetCurrentSlots(c *gin.Context) {
	now := time.Now()

	var slots []models.WorkshopTimeSlot
	err := in.DB.WithContext(c.Request.Context()).
		Preload("Facilitator").
		Preload("Workshop").
		Where("slot_start <= ? AND slot_end >= ?", now, now).
		Find(&slots).Error

	if err != nil {
		config.Log.Error("Database hatası", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(slots) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "Şu anda aktif slot yok",
			"slots":   []models.TimeSlotResponse{},
			"total":   0,
		})
		return
	}

	var response []models.TimeSlotResponse
	for _, slot := range slots {
		response = append(response, models.TimeSlotResponse{
			SlotID:    slot.SlotID,
			SlotStart: slot.SlotStart,
			SlotEnd:   slot.SlotEnd,
			SlotOrder: slot.SlotOrder,
			Facilitator: models.FacilitatorResponse{
				FacilitatorID: slot.Facilitator.FacilitatorID,
				Name:          slot.Facilitator.Name,
				Topic:         slot.Facilitator.Topic,
				Tags:          slot.Facilitator.Tags,
				TopicDetails:  slot.Facilitator.TopicDetails,
				Photograph:    slot.Facilitator.Photograph,
			},
		})
	}

	// Workshop bilgilerini de ekle
	var workshopInfo []gin.H
	for _, slot := range slots {
		workshopInfo = append(workshopInfo, gin.H{
			"workshop_id":   slot.Workshop.WorkshopID,
			"workshop_name": slot.Workshop.WorkshopName,
			"slot":          response[len(workshopInfo)],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"active_workshops": workshopInfo,
		"total":            len(workshopInfo),
	})
}

func GetUpcomingSlots(c *gin.Context) {
	limit := 5
	now := time.Now()

	var slots []models.WorkshopTimeSlot
	err := in.DB.WithContext(c.Request.Context()).
		Preload("Facilitator").
		Preload("Workshop").
		Where("slot_start > ?", now).
		Order("slot_start ASC").
		Limit(limit).
		Find(&slots).Error

	if err != nil {
		config.Log.Error("Database hatası", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(slots) == 0 {
		config.Log.Info("Yaklaşan slot bulunamadı")
		c.JSON(http.StatusOK, gin.H{
			"message":        "Gelecekte bir workshop görünmüyor, ilginiz için teşekkür ederiz.",
			"upcoming_slots": []models.UpcomingSlotResponse{},
			"total":          0,
		})
		return
	}

	var response []models.UpcomingSlotResponse
	for _, slot := range slots {
		timeUntil := slot.SlotStart.Sub(now)
		timeText := formatDuration(timeUntil)

		response = append(response, models.UpcomingSlotResponse{
			SlotID:       slot.SlotID,
			WorkshopName: slot.Workshop.WorkshopName,
			SlotStart:    slot.SlotStart,
			SlotEnd:      slot.SlotEnd,
			Facilitator: models.FacilitatorResponse{
				FacilitatorID: slot.Facilitator.FacilitatorID,
				Name:          slot.Facilitator.Name,
				Topic:         slot.Facilitator.Topic,
				Tags:          slot.Facilitator.Tags,
				TopicDetails:  slot.Facilitator.TopicDetails,
				Photograph:    slot.Facilitator.Photograph,
			},
			TimeUntilStart: timeText,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"upcoming_slots": response,
		"total":          len(response),
	})
}

func AddDelayToWorkshop(c *gin.Context) {
	workshopID := c.Param("id")

	type DelayRequest struct {
		DelayMinutes int `json:"delay_minutes" binding:"required"` // 5, 10, -5 vs.
	}

	var req DelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()

	result := in.DB.Model(&models.WorkshopTimeSlot{}).
		Where("workshop_id = ? AND slot_end > ?", workshopID, now).
		Updates(map[string]interface{}{
			"slot_start": gorm.Expr("slot_start + interval '1 minute' * ?", req.DelayMinutes),
			"slot_end":   gorm.Expr("slot_end + interval '1 minute' * ?", req.DelayMinutes),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		config.Log.Error("Slot'lar güncellenirken hata", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot'lar güncellenemedi"})
		return
	}

	updatedCount := result.RowsAffected

	var message string
	if req.DelayMinutes > 0 {
		message = fmt.Sprintf("Workshop %d dakika ertelendi. %d slot güncellendi.", req.DelayMinutes, updatedCount)
	} else if req.DelayMinutes < 0 {
		message = fmt.Sprintf("Workshop %d dakika erkene alındı. %d slot güncellendi.", -req.DelayMinutes, updatedCount)
	} else {
		message = "Gecikme sıfırlandı."
	}

	config.Log.Info(message)
	c.JSON(http.StatusOK, gin.H{
		"message":       message,
		"delay_minutes": req.DelayMinutes,
		"updated_slots": updatedCount,
	})
}

// Helper function
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 1 {
		return "Şimdi başlıyor"
	} else if minutes < 60 {
		return fmt.Sprintf("%d dakika sonra", minutes)
	} else {
		hours := minutes / 60
		remainingMinutes := minutes % 60
		if remainingMinutes == 0 {
			return fmt.Sprintf("%d saat sonra", hours)
		}
		return fmt.Sprintf("%d saat %d dakika sonra", hours, remainingMinutes)
	}
}

func GetAllWorkshops(c *gin.Context) {
	var workshops []models.Workshops
	result := in.DB.WithContext(c.Request.Context()).
		Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
			return db.Order("slot_order ASC")
		}).
		Preload("TimeSlots.Facilitator").
		Find(&workshops)

	if result.Error != nil {
		config.Log.Error("Workshop'lar alınırken hata oluştu", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workshops"})
		return
	}

	if len(workshops) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Workshop bulunamadı",
			"workshops": []models.Workshops{},
			"total":     0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workshops": workshops,
		"total":     len(workshops),
	})
	config.Log.Info("Tüm workshop'lar alındı", zap.Int("total", len(workshops)))
}

func DeleteWorkshop(c *gin.Context) {
	workshopID := c.Param("id")

	if workshopID == "" {
		config.Log.Warn("Workshop ID parametresi boş")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workshop ID gerekli"})
		return
	}

	var workshopIDInt uint
	if _, err := fmt.Sscanf(workshopID, "%d", &workshopIDInt); err != nil {
		config.Log.Warn("Geçersiz workshop ID formatı", zap.String("id", workshopID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz workshop ID"})
		return
	}

	// 3. Workshop var mı kontrol et
	var workshop models.Workshops
	if err := in.DB.First(&workshop, workshopID).Error; err != nil {
		config.Log.Warn("Silinmek istenen workshop bulunamadı", zap.String("id", workshopID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		return
	}

	// 4. Transaction başlat
	tx := in.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		config.Log.Error("Transaction başlatılamadı", zap.Error(tx.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "İşlem başlatılamadı"})
		return
	}

	// 5. Önce slot'ları sil
	deleteSlots := tx.Where("workshop_id = ?", workshopID).Delete(&models.WorkshopTimeSlot{})
	if deleteSlots.Error != nil {
		tx.Rollback()
		config.Log.Error("Slot'lar silinirken hata", zap.Error(deleteSlots.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot'lar silinemedi"})
		return
	}

	deletedSlotCount := deleteSlots.RowsAffected

	// 6. Sonra workshop'u sil
	if err := tx.Delete(&workshop).Error; err != nil {
		tx.Rollback()
		config.Log.Error("Workshop silinirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Workshop silinemedi"})
		return
	}

	// 7. Transaction'ı tamamla
	if err := tx.Commit().Error; err != nil {
		config.Log.Error("Transaction commit edilemedi", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "İşlem tamamlanamadı"})
		return
	}

	config.Log.Info("Workshop silindi", zap.String("id", workshopID), zap.Int64("deleted_slots", deletedSlotCount))
	c.JSON(http.StatusOK, gin.H{
		"message":       "Workshop ve slot'ları başarıyla silindi",
		"workshop_id":   workshop.WorkshopID,
		"workshop_name": workshop.WorkshopName,
		"deleted_slots": deletedSlotCount,
	})
}

func DeleteSlots(c *gin.Context) {
	slotID := c.Param("id")

	if slotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Slot ID gerekli",
		})
		config.Log.Warn("Slot ID parametresi boş")
		return
	}

	// Önce slot var mı kontrol et
	var slot models.WorkshopTimeSlot
	if err := in.DB.First(&slot, "slot_id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Silinmek istenen slot bulunamadı",
		})
		config.Log.Warn("Silinmek istenen slot bulunamadı", zap.String("id", slotID))
		return
	}

	// Slot'u sil
	result := in.DB.Delete(&slot)
	if result.Error != nil {
		config.Log.Error("Slot silinirken hata", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot silinemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Slot başarıyla silindi",
		"slot_id": slotID,
	})
	config.Log.Info("Slot silindi", zap.String("id", slotID))
}

func UpdateWorkshops(c *gin.Context) {
	workshopID := c.Param("id")

	if workshopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Workshop ID gerekli",
		})
		config.Log.Warn("Workshop ID boş.")
		return
	}

	type UpdateWorkshopRequest struct {
		WorkshopName string    `json:"workshop_name"`
		WorkshopDate time.Time `json:"workshop_date"`
	}

	var req UpdateWorkshopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		config.Log.Warn("JSON parse hatası", zap.Error(err))
		return
	}

	var workshop models.Workshops
	if err := in.DB.First(&workshop, workshopID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		config.Log.Warn("Güncellenecek workshop bulunamadı", zap.String("id", workshopID))
		return
	}

	// Güncelle
	updateData := map[string]interface{}{}

	if req.WorkshopName != "" {
		updateData["workshop_name"] = req.WorkshopName
	}
	if !req.WorkshopDate.IsZero() {
		updateData["workshop_date"] = req.WorkshopDate
	}

	if err := in.DB.Model(&workshop).Updates(updateData).Error; err != nil {
		config.Log.Error("Workshop güncellenirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Workshop güncellenemedi"})
		return
	}

	config.Log.Info("Workshop güncellendi", zap.String("id", workshopID), zap.String("name", req.WorkshopName))
	c.JSON(http.StatusOK, gin.H{
		"message":  "Workshop başarıyla güncellendi",
		"workshop": workshop,
	})
}

func UpdateTimeSlot(c *gin.Context) {
	slotID := c.Param("id")

	// 1. Parametre kontrolü
	if slotID == "" {
		config.Log.Warn("Slot ID boş")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID gerekli"})
		return
	}

	// 2. Request struct
	type UpdateSlotRequest struct {
		FacilitatorID *uint      `json:"facilitator_id"` // Pointer = opsiyonel
		SlotStart     *time.Time `json:"slot_start"`
		SlotEnd       *time.Time `json:"slot_end"`
		SlotOrder     *int       `json:"slot_order"`
	}

	var req UpdateSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		config.Log.Warn("Geçersiz JSON formatı", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı"})
		return
	}

	// 3. Slot var mı kontrol et
	var slot models.WorkshopTimeSlot
	if err := in.DB.First(&slot, slotID).Error; err != nil {
		config.Log.Warn("Güncellenecek slot bulunamadı", zap.String("id", slotID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot bulunamadı"})
		return
	}

	targetStart := slot.SlotStart
	if req.SlotStart != nil {
		targetStart = *req.SlotStart
	}

	targetEnd := slot.SlotEnd
	if req.SlotEnd != nil {
		targetEnd = *req.SlotEnd
	}

	if req.SlotStart != nil || req.SlotEnd != nil {
		var conflictCount int64
		err := in.DB.Model(&models.WorkshopTimeSlot{}).
			Where("workshop_id = ? AND slot_id <> ? AND slot_start = ? AND slot_end = ?",
				slot.WorkshopID, slot.SlotID, targetStart, targetEnd).
			Count(&conflictCount).Error

		if err != nil {
			config.Log.Error("Çakışma kontrolü sırasında DB hatası", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sistem hatası"})
			return
		}

		if conflictCount > 0 {
			config.Log.Warn("Slot çakışması engellendi", zap.Uint("workshop_id", slot.WorkshopID), zap.Time("start", targetStart))
			c.JSON(http.StatusConflict, gin.H{"error": "Bu zaman aralığında bu workshop için zaten bir slot mevcut. Güncelleme reddedildi."})
			return
		}
	}

	updates := make(map[string]interface{})

	if req.FacilitatorID != nil {
		var facilitator models.Facilitators
		if err := in.DB.First(&facilitator, *req.FacilitatorID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Konuşmacı bulunamadı"})
			return
		}
		updates["facilitator_id"] = *req.FacilitatorID
	}

	if req.SlotStart != nil {
		updates["slot_start"] = *req.SlotStart
	}

	if req.SlotEnd != nil {
		updates["slot_end"] = *req.SlotEnd
	}

	if req.SlotOrder != nil {
		updates["slot_order"] = *req.SlotOrder
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Güncellenecek alan bulunamadı"})
		return
	}

	if req.SlotStart != nil && req.SlotEnd != nil {
		if req.SlotEnd.Before(*req.SlotStart) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bitiş zamanı başlangıçtan önce olamaz"})
			return
		}
	}

	if err := in.DB.Model(&slot).Updates(updates).Error; err != nil {
		config.Log.Error("Slot güncellenirken hata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot güncellenemedi"})
		return
	}

	if err := in.DB.Preload("Facilitator").Preload("Workshop").First(&slot, slotID).Error; err != nil {
		config.Log.Error("Güncel slot alınamadı", zap.Error(err))
	}

	config.Log.Info("Slot güncellendi", zap.String("id", slotID))
	c.JSON(http.StatusOK, gin.H{
		"message": "Slot başarıyla güncellendi",
		"slot":    slot,
	})
}

func GetCurrentSlotInWorkshop(c *gin.Context) {
	workshopID := c.Param("id")

	if workshopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workshop ID'si gerekli"})
		config.Log.Warn("Workshop ID Parametresi boş")
		return
	}

	var workshop models.Workshops
	err := in.DB.WithContext(c.Request.Context()).Preload("TimeSlots").Preload("TimeSlots.Facilitator").
		Where("slot_start <= ? AND slot_end >= ?", time.Now(), time.Now()).
		First(&workshop, workshopID).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı veya şu an aktif bir slot yok"})
		config.Log.Warn("Workshop bulunamadı", zap.String("id", workshopID))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workshop başarıyla alındı", "workshop": workshop})
	config.Log.Info("Workshop alındı", zap.String("id", workshopID))
}
