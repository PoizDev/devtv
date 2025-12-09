package controllers

import (
	"devtv/in"
	"devtv/models"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/jeanphorn/log4go"
	"gorm.io/gorm"
)

// --- WebSocket Altyapısı ---

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	workshopCurrentSlotManagers = make(map[string]*ClientManager)
	workshopCurrentSlotLock     sync.RWMutex
)

// ClientManager - Bağlantıları güvenli bir şekilde yönetmek için
type ClientManager struct {
	clients map[*websocket.Conn]bool
	lock    sync.RWMutex
}

func (cm *ClientManager) Add(conn *websocket.Conn) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.clients[conn] = true
}

func (cm *ClientManager) Remove(conn *websocket.Conn) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	if _, ok := cm.clients[conn]; ok {
		delete(cm.clients, conn)
		conn.Close()
	}
}

// Güvenli bir şekilde tüm istemcilere mesaj gönderir
func (cm *ClientManager) Broadcast(message interface{}) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	for client := range cm.clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Error("WebSocket yazma hatası, client siliniyor: %v", err)
			client.Close()
			// Not: Loop içinde map'ten silmek güvenli olmadığından
			// burada sadece close ediyoruz, clean-up goroutine veya
			// bir sonraki cycle'da silinebilir. Ancak basitlik adına
			// burada go routine ile silme tetiklenebilir.
			go func(c *websocket.Conn) {
				cm.lock.Lock()
				delete(cm.clients, c)
				cm.lock.Unlock()
			}(client)
		}
	}
}

// Her servis için ayrı bir Manager oluşturuyoruz
var (
	currentSlotsManager  = ClientManager{clients: make(map[*websocket.Conn]bool)}
	upcomingSlotsManager = ClientManager{clients: make(map[*websocket.Conn]bool)}
	sponsorsManager      = ClientManager{clients: make(map[*websocket.Conn]bool)}
	// ID bazlı workshoplar için map içinde manager (WorkshopID -> Manager)
	workshopSchedManagers = make(map[string]*ClientManager)
	workshopSchedLock     sync.RWMutex
)

// --- Başlatıcı (Main.go içinde çağrılmalı veya init ile otomatik başlar) ---

func init() {
	// Arka plan işlerini başlat
	go startCurrentSlotsBroadcaster()
	go startUpcomingSlotsBroadcaster()
	go startSponsorsBroadcaster()
}

// --- Controller Fonksiyonları ---

// GetCurrentSlotsWS - Sadece kullanıcıyı havuza ekler, sorgu yapmaz
func GetCurrentSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}

	currentSlotsManager.Add(ws)
	log.Info("Yeni WebSocket bağlantısı: GetCurrentSlots")

	// Bağlantı kopana kadar dinle (Ping/Pong için)
	reader(ws, &currentSlotsManager)
}

// GetUpcomingSlotsWS
func GetUpcomingSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}

	upcomingSlotsManager.Add(ws)
	log.Info("Yeni WebSocket bağlantısı: GetUpcomingSlots")
	reader(ws, &upcomingSlotsManager)
}

// GetSponsorsWS
func GetSponsorsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}

	sponsorsManager.Add(ws)
	log.Info("Yeni WebSocket bağlantısı: GetSponsors")
	reader(ws, &sponsorsManager)
}

// GetWorkshopScheduleWS
func GetWorkshopScheduleWS(c *gin.Context) {
	workshopID := c.Param("id")
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// İlgili workshop için manager var mı? Yoksa yarat.
	workshopSchedLock.Lock()
	if _, exists := workshopSchedManagers[workshopID]; !exists {
		workshopSchedManagers[workshopID] = &ClientManager{clients: make(map[*websocket.Conn]bool)}
		// Bu ID için özel yayıncı başlat
		go startSpecificWorkshopBroadcaster(workshopID)
	}
	manager := workshopSchedManagers[workshopID]
	workshopSchedLock.Unlock()

	manager.Add(ws)
	log.Info("Yeni WS: GetWorkshopSchedule - ID: ", workshopID)
	reader(ws, manager)
}

// GetCurrentSlotInWorkshopWS (Eski kodunuzda vardı, buraya da ekledim)
// Not: Bu fonksiyon da GetWorkshopScheduleWS mantığıyla ID bazlı çalışmalı.
// Basitlik adına yukarıdaki yapıyı kullanabilirsiniz.

// --- Helper: Okuma Döngüsü ---
func reader(ws *websocket.Conn, cm *ClientManager) {
	defer cm.Remove(ws)
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}
}

// --- ARKA PLAN YAYINCILARI (BROADCASTERS) ---
// Bu fonksiyonlar veritabanını sadece 1 kez sorgular ve binlerce kişiye dağıtır.

func startCurrentSlotsBroadcaster() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		// Eğer hiç client yoksa sorgu yapma (Performans tasarrufu)
		currentSlotsManager.lock.RLock()
		count := len(currentSlotsManager.clients)
		currentSlotsManager.lock.RUnlock()
		if count == 0 {
			continue
		}

		now := time.Now()
		var slots []models.WorkshopTimeSlot
		// DB Sorgusu (Tek sefer çalışır)
		err := in.DB.
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start <= ? AND slot_end >= ?", now, now).
			Find(&slots).Error

		if err != nil {
			log.Error("Broadcaster DB Error: ", err)
			continue
		}

		// Veriyi hazırla
		var data gin.H
		if len(slots) == 0 {
			data = gin.H{
				"message":   "Şu anda aktif slot yok",
				"slots":     []models.TimeSlotResponse{},
				"total":     0,
				"timestamp": time.Now(),
			}
		} else {
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
			data = gin.H{
				"active_workshops": workshopInfo,
				"total":            len(workshopInfo),
				"timestamp":        time.Now(),
			}
		}

		// Herkese gönder
		currentSlotsManager.Broadcast(data)
	}
}

func startUpcomingSlotsBroadcaster() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		upcomingSlotsManager.lock.RLock()
		if len(upcomingSlotsManager.clients) == 0 {
			upcomingSlotsManager.lock.RUnlock()
			continue
		}
		upcomingSlotsManager.lock.RUnlock()

		now := time.Now()
		var slots []models.WorkshopTimeSlot
		err := in.DB.
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start > ?", now).
			Order("slot_start").
			Find(&slots).Error

		if err != nil {
			log.Error("Upcoming DB Error: ", err)
			continue
		}

		var data gin.H
		if len(slots) == 0 {
			data = gin.H{
				"message":        "Gelecekte bir workshop görünmüyor",
				"upcoming_slots": []models.UpcomingSlotResponse{},
				"total":          0,
				"timestamp":      time.Now(),
			}
		} else {
			var response []models.UpcomingSlotResponse
			for _, slot := range slots {
				timeUntil := slot.SlotStart.Sub(now)
				// formatDuration fonksiyonunun tanımlı olduğunu varsayıyorum
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
			data = gin.H{
				"upcoming_slots": response,
				"total":          len(response),
				"timestamp":      time.Now(),
			}
		}
		upcomingSlotsManager.Broadcast(data)
	}
}

func startSponsorsBroadcaster() {
	ticker := time.NewTicker(10 * time.Second) // Sponsorlar az değişir, süreyi artırdım
	for range ticker.C {
		sponsorsManager.lock.RLock()
		if len(sponsorsManager.clients) == 0 {
			sponsorsManager.lock.RUnlock()
			continue
		}
		sponsorsManager.lock.RUnlock()

		var sponsors []models.Sponsors
		err := in.DB.Find(&sponsors).Error
		if err != nil {
			continue
		}

		data := gin.H{
			"sponsors":  sponsors,
			"total":     len(sponsors),
			"timestamp": time.Now(),
		}
		sponsorsManager.Broadcast(data)
	}
}

// Belirli bir ID için çalışan özel broadcaster
func startSpecificWorkshopBroadcaster(workshopID string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Manager'a erişim için pointer al
	workshopSchedLock.RLock()
	manager, exists := workshopSchedManagers[workshopID]
	workshopSchedLock.RUnlock()

	if !exists {
		return
	}

	for range ticker.C {
		manager.lock.RLock()
		count := len(manager.clients)
		manager.lock.RUnlock()

		// Eğer kimse bu odayı dinlemiyorsa döngüden çık ve goroutine'i öldür (Memory leak önleme)
		if count == 0 {
			workshopSchedLock.Lock()
			delete(workshopSchedManagers, workshopID)
			workshopSchedLock.Unlock()
			log.Info("Broadcaster kapatılıyor: ", workshopID)
			return
		}

		// --- DB Mantığı (Eski koddan alındı) ---
		var workshop models.Workshops
		err := in.DB.
			Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
				return db.Order("slot_order ASC")
			}).
			Preload("TimeSlots.Faciliator").
			First(&workshop, workshopID).Error

		if err != nil {
			manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
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
			CurrentSlot:  currentSlot,
			AllSlots:     allSlots,
			TotalSlots:   len(allSlots),
		}

		manager.Broadcast(response)
	}
}

// Helper function (eğer başka dosyada yoksa buraya ekleyin)
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	if h > 0 {
		return fmt.Sprintf("%d saat %d dk", h, m)
	}
	return fmt.Sprintf("%d dk", m)
}

func startWorkshopCurrentSlotBroadcaster(workshopID string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Manager'a referans al
	workshopCurrentSlotLock.RLock()
	manager, exists := workshopCurrentSlotManagers[workshopID]
	workshopCurrentSlotLock.RUnlock()

	if !exists {
		return
	}

	for range ticker.C {
		// Dinleyen kimse yoksa döngüyü ve goroutine'i kapat (Memory Leak önlemi)
		manager.lock.RLock()
		count := len(manager.clients)
		manager.lock.RUnlock()

		if count == 0 {
			workshopCurrentSlotLock.Lock()
			delete(workshopCurrentSlotManagers, workshopID)
			workshopCurrentSlotLock.Unlock()
			log.Info("Workshop Current Slot yayını kapatıldı (kimse yok): ", workshopID)
			return
		}

		// --- Veritabanı ve Mantık İşlemleri ---
		now := time.Now()
		var workshop models.Workshops

		// DB sorgusu optimize edildi
		err := in.DB.
			Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
				return db.Order("slot_order ASC")
			}).
			Preload("TimeSlots.Faciliator").
			First(&workshop, workshopID).Error

		if err != nil {
			manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
			continue
		}

		// Şu anki ve sonraki slotu bul
		var currentSlot *models.TimeSlotResponse
		var nextSlot *models.TimeSlotResponse

		for i, slot := range workshop.TimeSlots {
			// Aktif slot mu?
			if now.After(slot.SlotStart) && now.Before(slot.SlotEnd) {
				currentSlot = &models.TimeSlotResponse{
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

				// Sonraki slotu al
				if i+1 < len(workshop.TimeSlots) {
					nxt := workshop.TimeSlots[i+1]
					nextSlot = &models.TimeSlotResponse{
						SlotID:    nxt.SlotID,
						SlotStart: nxt.SlotStart,
						SlotEnd:   nxt.SlotEnd,
						SlotOrder: nxt.SlotOrder,
						Faciliator: models.FaciliatorResponse{
							FaciliatorID: nxt.Faciliator.FaciliatorID,
							Name:         nxt.Faciliator.Name,
							Topic:        nxt.Faciliator.Topic,
							TopicDetails: nxt.Faciliator.TopicDetails,
							Photograph:   nxt.Faciliator.Photograph,
						},
					}
				}
				break
			}
		}

		// Eğer aktif slot bulunamadıysa, gelecekteki en yakın slotu "next" olarak ayarla
		if currentSlot == nil {
			for _, slot := range workshop.TimeSlots {
				if now.Before(slot.SlotStart) {
					nextSlot = &models.TimeSlotResponse{
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
					break
				}
			}
		}

		// Response Hazırla
		response := gin.H{
			"workshop_id":   workshop.WorkshopID,
			"workshop_name": workshop.WorkshopName,
			"timestamp":     now,
		}

		if currentSlot != nil {
			remainingTime := currentSlot.SlotEnd.Sub(now)
			response["current_slot"] = currentSlot
			response["status"] = "active"
			response["remaining_minutes"] = int(remainingTime.Minutes())
			response["remaining_time"] = formatDuration(remainingTime)
		} else {
			response["current_slot"] = nil
			response["status"] = "waiting"
			response["message"] = "Şu anda aktif slot yok"
		}

		if nextSlot != nil {
			timeUntilStart := nextSlot.SlotStart.Sub(now)
			response["next_slot"] = nextSlot
			response["time_until_next"] = formatDuration(timeUntilStart)
		} else {
			response["next_slot"] = nil
			response["message"] = "Sırada slot yok"
		}

		// Bu odaya bağlı tüm kullanıcılara tek seferde gönder
		manager.Broadcast(response)
	}
}

// GetCurrentSlotInWorkshopWS - Belirli bir workshop'un anlık durumunu stream eder
func GetCurrentSlotInWorkshopWS(c *gin.Context) {
	workshopID := c.Param("id")

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade hatası: ", err)
		return
	}

	// Bu workshop ID için bir yayıncı (broadcaster) var mı kontrol et
	workshopCurrentSlotLock.Lock()
	if _, exists := workshopCurrentSlotManagers[workshopID]; !exists {
		// Yoksa yeni bir kanal oluştur ve yayıncıyı başlat
		workshopCurrentSlotManagers[workshopID] = &ClientManager{clients: make(map[*websocket.Conn]bool)}
		go startWorkshopCurrentSlotBroadcaster(workshopID)
	}
	manager := workshopCurrentSlotManagers[workshopID]
	workshopCurrentSlotLock.Unlock()

	// Kullanıcıyı havuza ekle
	manager.Add(ws)
	log.Info(fmt.Sprintf("Yeni WS Bağlantısı: GetCurrentSlotInWorkshopWS - Workshop ID: %s", workshopID))

	// Bağlantı kopana kadar dinle
	reader(ws, manager)
}
