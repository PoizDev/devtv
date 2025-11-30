package controllers

import (
	"devtv/in"
	"devtv/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/jeanphorn/log4go"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Üretimde domain kontrol et
	},
}

// WebSocket bağlantılarını yönet
var clients = make(map[*websocket.Conn]bool)

// GetCurrentSlotsWS - WebSocket ile şu anda aktif slot'ları stream et
func GetCurrentSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Info("Yeni WebSocket bağlantısı: GetCurrentSlots")

	// Her 5 saniyede bir güncelle
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				delete(clients, ws)
				log.Info("WebSocket bağlantısı kapatıldı: GetCurrentSlots")
				return
			}
		}
	}()

	for range ticker.C {
		now := time.Now()

		var slots []models.WorkshopTimeSlot
		err := in.DB.
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start <= ? AND slot_end >= ?", now, now).
			Find(&slots).Error

		if err != nil {
			log.Error("Database hatası: ", err)
			ws.WriteJSON(gin.H{"error": err.Error()})
			continue
		}

		if len(slots) == 0 {
			ws.WriteJSON(gin.H{
				"message":   "Şu anda aktif slot yok",
				"slots":     []models.TimeSlotResponse{},
				"total":     0,
				"timestamp": time.Now(),
			})
			continue
		}

		var response []models.TimeSlotResponse
		for _, slot := range slots {
			response = append(response, models.TimeSlotResponse{
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
			})
		}

		var workshopInfo []gin.H
		for i, slot := range slots {
			workshopInfo = append(workshopInfo, gin.H{
				"workshop_id":   slot.Workshop.WorkshopID,
				"workshop_name": slot.Workshop.WorkshopName,
				"slot":          response[i],
			})
		}

		ws.WriteJSON(gin.H{
			"active_workshops": workshopInfo,
			"total":            len(workshopInfo),
			"timestamp":        time.Now(),
		})
	}
}

// GetWorkshopScheduleWS - WebSocket ile workshop programını stream et
func GetWorkshopScheduleWS(c *gin.Context) {
	workshopID := c.Param("id")

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Info("Yeni WebSocket bağlantısı: GetWorkshopSchedule - Workshop ID: ", workshopID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				delete(clients, ws)
				log.Info("WebSocket bağlantısı kapatıldı: GetWorkshopSchedule")
				return
			}
		}
	}()

	for range ticker.C {
		var workshop models.Workshops
		err := in.DB.
			Preload("TimeSlots", func(db any) any {
				return db.(interface{ Order(string) any }).Order("slot_order ASC")
			}).
			Preload("TimeSlots.Faciliator").
			First(&workshop, workshopID).Error

		if err != nil {
			ws.WriteJSON(gin.H{"error": "Workshop bulunamadı"})
			continue
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
				Faciliator: models.FaciliatorResponse{
					FaciliatorID: slot.Faciliator.FaciliatorID,
					Name:         slot.Faciliator.Name,
					Topic:        slot.Faciliator.Topic,
					TopicDetails: slot.Faciliator.TopicDetails,
					Photograph:   slot.Faciliator.Photograph,
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
			IsLive:       workshop.IsLive,
			CurrentSlot:  currentSlot,
			AllSlots:     allSlots,
			TotalSlots:   len(allSlots),
		}

		ws.WriteJSON(response)
	}
}
func GetUpcomingSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Info("Yeni WebSocket bağlantısı: GetUpcomingSlots")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				delete(clients, ws)
				log.Info("WebSocket bağlantısı kapatıldı: GetUpcomingSlots")
				return
			}
		}
	}()

	for range ticker.C {
		now := time.Now()

		var slots []models.WorkshopTimeSlot
		err := in.DB.
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start > ?", now).
			Order("slot_start ASC").
			Limit(5).
			Find(&slots).Error

		if err != nil {
			log.Error("Database hatası: ", err)
			ws.WriteJSON(gin.H{"error": err.Error()})
			continue
		}

		if len(slots) == 0 {
			ws.WriteJSON(gin.H{
				"message":        "Gelecekte bir workshop görünmüyor",
				"upcoming_slots": []models.UpcomingSlotResponse{},
				"total":          0,
				"timestamp":      time.Now(),
			})
			continue
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

		ws.WriteJSON(gin.H{
			"upcoming_slots": response,
			"total":          len(response),
			"timestamp":      time.Now(),
		})
	}
}

func GetSponsorsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Info("Yeni WebSocket bağlantısı: GetSponsors")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				delete(clients, ws)
				log.Info("WebSocket bağlantısı kapatıldı: GetSponsors")
				return
			}
		}
	}()

	for range ticker.C {
		var sponsors []models.Sponsors

		err := in.DB.Find(&sponsors).Error
		if err != nil {
			log.Error("Database hatası: ", err)
			ws.WriteJSON(gin.H{"error": err.Error()})
			continue
		}

		if len(sponsors) == 0 {
			ws.WriteJSON(gin.H{
				"message":   "Henüz sponsor eklenmemiş",
				"sponsors":  []models.Sponsors{},
				"total":     0,
				"timestamp": time.Now(),
			})
			continue
		}

		ws.WriteJSON(gin.H{
			"sponsors":  sponsors,
			"total":     len(sponsors),
			"timestamp": time.Now(),
		})
	}
}
