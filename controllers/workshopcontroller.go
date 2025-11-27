package controllers

import (
	"devtv/in"
	"devtv/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
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
		IsLive:       false,
		TimeSlots:    req.TimeSlots, // Boş array bile olsa sorun yok
	}

	err := in.DB.Create(&workshop).Error
	if err != nil {
		log.Error("Workshop oluşturulurken hata: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Workshop oluşturulamadı"})
		return
	}

	message := fmt.Sprintf("Workshop oluşturuldu: %s", workshop.WorkshopName)
	if len(req.TimeSlots) > 0 {
		message += fmt.Sprintf(" (%d slot eklendi)", len(req.TimeSlots))
	} else {
		message += " (slot'sız)"
	}

	log.Info(message)
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

	// Workshop var mı kontrol et
	var workshop models.Workshops
	if err := in.DB.First(&workshop, workshopID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		return
	}

	// Her slot için facilitator kontrolü yap
	for _, slot := range req.TimeSlots {
		var facilitator models.Faciliators
		if err := in.DB.First(&facilitator, slot.FaciliatorID).Error; err != nil {
			log.Error("Facilitator ID'si %d olan kişi bulunamadı.", slot.FaciliatorID)
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Facilitator ID'si %d olan kişi bulunamadı", slot.FaciliatorID)})
			return
		}
	}

	// Mevcut en büyük slot_order'ı bul
	var maxOrder int
	in.DB.Model(&models.WorkshopTimeSlot{}).
		Where("workshop_id = ?", workshopID).
		Select("COALESCE(MAX(slot_order), 0)").
		Scan(&maxOrder)

	// Yeni slot'lara sıralı order ekle
	for i := range req.TimeSlots {
		req.TimeSlots[i].WorkshopID = workshop.WorkshopID
		req.TimeSlots[i].SlotOrder = maxOrder + i + 1
	}

	// Slot'ları kaydet
	if err := in.DB.Create(&req.TimeSlots).Error; err != nil {
		log.Error("Slot'lar eklenirken hata: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slot'lar eklenemedi"})
		return
	}

	log.Info(fmt.Sprintf("Workshop ID %s'e %d yeni slot eklendi", workshopID, len(req.TimeSlots)))
	c.JSON(http.StatusCreated, gin.H{
		"message":     "Slot'lar başarıyla eklendi",
		"added_slots": len(req.TimeSlots),
		"slots":       req.TimeSlots,
	})
}

// GetWorkshopSchedule - Belirli bir workshop'un programını getir
func GetWorkshopSchedule(c *gin.Context) {
	workshopID := c.Param("id")

	var workshop models.Workshops
	err := in.DB.
		Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
			return db.Order("slot_order ASC")
		}).
		Preload("TimeSlots.Faciliator").
		First(&workshop, workshopID).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workshop bulunamadı"})
		return
	}

	// Şu anki slot'u bul
	now := time.Now()
	var currentSlot *models.TimeSlotResponse
	var allSlots []models.TimeSlotResponse

	for _, slot := range workshop.TimeSlots {
		slotResponse := models.TimeSlotResponse{
			SlotID:    slot.SlotID,
			SlotStart: slot.SlotStart,
			SlotEnd:   slot.SlotEnd,
			SlotOrder: slot.SlotOrder,
			Faciliator: models.FaciliatorResponse{
				FaciliatorID: slot.Faciliator.FaciliatorID,
				Name:         slot.Faciliator.Name,
				Topic:        slot.Faciliator.Topic,
				TopicDetails: slot.Faciliator.TopicDetails,
				Photograph:   slot.Faciliator.Photograph,
			},
		}

		// Şu anda aktif mi?
		if now.After(slot.SlotStart) && now.Before(slot.SlotEnd) {
			currentSlot = &slotResponse
		}

		allSlots = append(allSlots, slotResponse)
	}

	response := models.WorkshopScheduleResponse{
		WorkshopID:   workshop.WorkshopID,
		WorkshopName: workshop.WorkshopName,
		WorkshopDate: workshop.WorkshopDate,
		IsLive:       workshop.IsLive,
		CurrentSlot:  currentSlot,
		AllSlots:     allSlots,
		TotalSlots:   len(allSlots),
	}

	c.JSON(http.StatusOK, response)
}

// GetCurrentSlot - Şu anda aktif olan slot'u getir
func GetCurrentSlot(c *gin.Context) {
	now := time.Now()

	var slot models.WorkshopTimeSlot
	err := in.DB.
		Preload("Faciliator").
		Preload("Workshop").
		Where("slot_start <= ? AND slot_end >= ?", now, now).
		First(&slot).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Şu anda aktif slot yok",
			"slot":    nil,
		})
		return
	}

	response := models.TimeSlotResponse{
		SlotID:    slot.SlotID,
		SlotStart: slot.SlotStart,
		SlotEnd:   slot.SlotEnd,
		SlotOrder: slot.SlotOrder,
		Faciliator: models.FaciliatorResponse{
			FaciliatorID: slot.Faciliator.FaciliatorID,
			Name:         slot.Faciliator.Name,
			Topic:        slot.Faciliator.Topic,
			TopicDetails: slot.Faciliator.TopicDetails,
			Photograph:   slot.Faciliator.Photograph,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"slot":          response,
		"workshop_name": slot.Workshop.WorkshopName,
	})
}

// GetUpcomingSlots - Sıradaki slot'ları getir
func GetUpcomingSlots(c *gin.Context) {
	limit := 5
	now := time.Now()

	var slots []models.WorkshopTimeSlot
	err := in.DB.
		Preload("Faciliator").
		Preload("Workshop").
		Where("slot_start > ?", now).
		Order("slot_start ASC").
		Limit(limit).
		Find(&slots).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			Faciliator: models.FaciliatorResponse{
				FaciliatorID: slot.Faciliator.FaciliatorID,
				Name:         slot.Faciliator.Name,
				Topic:        slot.Faciliator.Topic,
				TopicDetails: slot.Faciliator.TopicDetails,
				Photograph:   slot.Faciliator.Photograph,
			},
			TimeUntilStart: timeText,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"upcoming_slots": response,
		"total":          len(response),
	})
}

// AddDelayToWorkshop - Belirli bir workshop'a gecikme ekle
// SADECE bu workshop'un gelecek slot'larını günceller
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

	// SADECE bu workshop'un gelecek slot'larını güncelle
	result := in.DB.Exec(`
		UPDATE workshop_time_slots 
		SET slot_start = slot_start + INTERVAL '? minutes',
		    slot_end = slot_end + INTERVAL '? minutes',
		    updated_at = NOW()
		WHERE workshop_id = ? 
		  AND slot_start > ?
	`, req.DelayMinutes, req.DelayMinutes, workshopID, now)

	if result.Error != nil {
		log.Error("Slot'lar güncellenirken hata: ", result.Error)
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

	log.Info(message)
	c.JSON(http.StatusOK, gin.H{
		"message":       message,
		"delay_minutes": req.DelayMinutes,
		"updated_slots": updatedCount,
	})
}

// SetWorkshopLive - Workshop'u canlı/kapalı yap
func SetWorkshopLive(c *gin.Context) {
	workshopID := c.Param("id")

	type LiveRequest struct {
		IsLive bool `json:"is_live"`
	}

	var req LiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := in.DB.Model(&models.Workshops{}).
		Where("workshop_id = ?", workshopID).
		Update("is_live", req.IsLive).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Güncelleme başarısız"})
		return
	}

	status := "kapatıldı"
	if req.IsLive {
		status = "canlı yayına alındı"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Workshop %s", status),
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
	result := in.DB.Find(&workshops)
	if result.Error != nil {
		log.Error("Workshop'lar alınırken hata oluştu: ", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workshops"})
		return
	}
	c.JSON(http.StatusOK, workshops)
	log.Info("Tüm workshop'lar alındı")
}
