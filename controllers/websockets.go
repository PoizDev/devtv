package controllers

import (
	"context"
	"devtv/in"
	"devtv/models"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/jeanphorn/log4go"
	"gorm.io/gorm"
)

// --- WebSocket Configuration ---
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// Handshake timeout ekle
	HandshakeTimeout: 10 * time.Second,
}

// --- Client Manager - Optimize edilmiş ---
type ClientManager struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan interface{}
	lock       sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewClientManager() *ClientManager {
	ctx, cancel := context.WithCancel(context.Background())
	cm := &ClientManager{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn, 100), // Buffer ekledik
		unregister: make(chan *websocket.Conn, 100), // Buffer ekledik
		broadcast:  make(chan interface{}, 1000),    // Büyük buffer
		ctx:        ctx,
		cancel:     cancel,
	}
	go cm.run()
	return cm
}

// run - Thread-safe client yönetimi (Tek goroutine)
func (cm *ClientManager) run() {
	ticker := time.NewTicker(30 * time.Second) // Ping ticker
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return

		case client := <-cm.register:
			cm.lock.Lock()
			cm.clients[client] = true
			cm.lock.Unlock()

		case client := <-cm.unregister:
			cm.lock.Lock()
			if _, ok := cm.clients[client]; ok {
				delete(cm.clients, client)
				client.Close()
			}
			cm.lock.Unlock()

		case message := <-cm.broadcast:
			cm.lock.RLock()
			clients := make([]*websocket.Conn, 0, len(cm.clients))
			for client := range cm.clients {
				clients = append(clients, client)
			}
			cm.lock.RUnlock()

			// Paralel gönderim (Goroutine pool kullanarak)
			var wg sync.WaitGroup
			for _, client := range clients {
				wg.Add(1)
				go func(c *websocket.Conn) {
					defer wg.Done()

					// Yazma timeout ekle
					c.SetWriteDeadline(time.Now().Add(5 * time.Second))

					if err := c.WriteJSON(message); err != nil {
						// Hata durumunda client'ı kaldır
						cm.unregister <- c
					}
				}(client)
			}
			wg.Wait()

		case <-ticker.C:
			// Ping gönder (connection health check)
			cm.lock.RLock()
			for client := range cm.clients {
				client.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := client.WriteMessage(websocket.PingMessage, nil); err != nil {
					cm.unregister <- client
				}
			}
			cm.lock.RUnlock()
		}
	}
}

func (cm *ClientManager) Add(conn *websocket.Conn) {
	cm.register <- conn
}

func (cm *ClientManager) Remove(conn *websocket.Conn) {
	cm.unregister <- conn
}

func (cm *ClientManager) Broadcast(message interface{}) {
	// Non-blocking send
	select {
	case cm.broadcast <- message:
	default:
		// Buffer dolu, eski mesajları at
		log.Warn("Broadcast buffer full, dropping message")
	}
}

func (cm *ClientManager) Count() int {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	return len(cm.clients)
}

func (cm *ClientManager) Shutdown() {
	cm.cancel()
}

// --- Global Managers ---
var (
	currentSlotsManager  *ClientManager
	upcomingSlotsManager *ClientManager
	sponsorsManager      *ClientManager

	workshopSchedManagers = make(map[string]*ClientManager)
	workshopSchedLock     sync.RWMutex

	workshopCurrentSlotManagers = make(map[string]*ClientManager)
	workshopCurrentSlotLock     sync.RWMutex

	initOnce sync.Once
)

func init() {
	initOnce.Do(func() {
		currentSlotsManager = NewClientManager()
		upcomingSlotsManager = NewClientManager()
		sponsorsManager = NewClientManager()

		// Broadcasters'ı başlat
		go startCurrentSlotsBroadcaster()
		go startUpcomingSlotsBroadcaster()
		go startSponsorsBroadcaster()
	})
}

// --- Controller Functions ---

func GetCurrentSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Log'u azalttık (sadece error)
		log.Error("WS upgrade error: ", err)
		return
	}

	// Pong handler
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	currentSlotsManager.Add(ws)

	// Okuma döngüsü - basitleştirilmiş
	go func() {
		defer currentSlotsManager.Remove(ws)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func GetUpcomingSlotsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WS upgrade error: ", err)
		return
	}

	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	upcomingSlotsManager.Add(ws)

	go func() {
		defer upcomingSlotsManager.Remove(ws)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func GetSponsorsWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WS upgrade error: ", err)
		return
	}

	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	sponsorsManager.Add(ws)

	go func() {
		defer sponsorsManager.Remove(ws)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func GetWorkshopScheduleWS(c *gin.Context) {
	workshopID := c.Param("id")
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Manager kontrolü - thread-safe
	workshopSchedLock.Lock()
	manager, exists := workshopSchedManagers[workshopID]
	if !exists {
		manager = NewClientManager()
		workshopSchedManagers[workshopID] = manager
		go startSpecificWorkshopBroadcaster(workshopID, manager)
	}
	workshopSchedLock.Unlock()

	manager.Add(ws)

	go func() {
		defer manager.Remove(ws)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func GetCurrentSlotInWorkshopWS(c *gin.Context) {
	workshopID := c.Param("id")
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WS upgrade error: ", err)
		return
	}

	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	workshopCurrentSlotLock.Lock()
	manager, exists := workshopCurrentSlotManagers[workshopID]
	if !exists {
		manager = NewClientManager()
		workshopCurrentSlotManagers[workshopID] = manager
		go startWorkshopCurrentSlotBroadcaster(workshopID, manager)
	}
	workshopCurrentSlotLock.Unlock()

	manager.Add(ws)

	go func() {
		defer manager.Remove(ws)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// --- BROADCASTERS - Optimize edilmiş ---

func startCurrentSlotsBroadcaster() {
	ticker := time.NewTicker(2 * time.Second) // 5s'den 2s'ye düşürdük
	defer ticker.Stop()

	var cachedData gin.H
	var cacheTime time.Time
	cacheDuration := 1 * time.Second // Cache süresi

	for range ticker.C {
		// Client yoksa işlem yapma
		if currentSlotsManager.Count() == 0 {
			continue
		}

		// Cache kontrolü
		if time.Since(cacheTime) < cacheDuration && cachedData != nil {
			currentSlotsManager.Broadcast(cachedData)
			continue
		}

		// DB Query - optimize edilmiş
		now := time.Now()
		var slots []models.WorkshopTimeSlot

		// Context ile timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := in.DB.WithContext(ctx).
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start <= ? AND slot_end >= ?", now, now).
			Find(&slots).Error
		cancel()

		if err != nil {
			// Log'u azalttık
			continue
		}

		// Response hazırla
		var data gin.H
		if len(slots) == 0 {
			data = gin.H{
				"message":   "Şu anda aktif slot yok",
				"slots":     []models.TimeSlotResponse{},
				"total":     0,
				"timestamp": now,
			}
		} else {
			response := make([]models.TimeSlotResponse, 0, len(slots))
			workshopInfo := make([]gin.H, 0, len(slots))

			for _, slot := range slots {
				slotResp := models.TimeSlotResponse{
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
				response = append(response, slotResp)

				workshopInfo = append(workshopInfo, gin.H{
					"workshop_id":   slot.Workshop.WorkshopID,
					"workshop_name": slot.Workshop.WorkshopName,
					"slot":          slotResp,
				})
			}

			data = gin.H{
				"active_workshops": workshopInfo,
				"total":            len(workshopInfo),
				"timestamp":        now,
			}
		}

		// Cache'e kaydet
		cachedData = data
		cacheTime = now

		// Broadcast
		currentSlotsManager.Broadcast(data)
	}
}

func startUpcomingSlotsBroadcaster() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var cachedData gin.H
	var cacheTime time.Time
	cacheDuration := 2 * time.Second

	for range ticker.C {
		if upcomingSlotsManager.Count() == 0 {
			continue
		}

		if time.Since(cacheTime) < cacheDuration && cachedData != nil {
			upcomingSlotsManager.Broadcast(cachedData)
			continue
		}

		now := time.Now()
		var slots []models.WorkshopTimeSlot

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := in.DB.WithContext(ctx).
			Preload("Faciliator").
			Preload("Workshop").
			Where("slot_start > ?", now).
			Order("slot_start ASC").
			Find(&slots).Error
		cancel()

		if err != nil {
			continue
		}

		var data gin.H
		if len(slots) == 0 {
			data = gin.H{
				"message":        "Gelecekte workshop yok",
				"upcoming_slots": []models.UpcomingSlotResponse{},
				"total":          0,
				"timestamp":      now,
			}
		} else {
			response := make([]models.UpcomingSlotResponse, 0, len(slots))

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

			data = gin.H{
				"upcoming_slots": response,
				"total":          len(response),
				"timestamp":      now,
			}
		}

		cachedData = data
		cacheTime = now
		upcomingSlotsManager.Broadcast(data)
	}
}

func startSponsorsBroadcaster() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var cachedData gin.H
	var cacheTime time.Time
	cacheDuration := 5 * time.Second

	for range ticker.C {
		if sponsorsManager.Count() == 0 {
			continue
		}

		if time.Since(cacheTime) < cacheDuration && cachedData != nil {
			sponsorsManager.Broadcast(cachedData)
			continue
		}

		var sponsors []models.Sponsors

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := in.DB.WithContext(ctx).Find(&sponsors).Error
		cancel()

		if err != nil {
			continue
		}

		data := gin.H{
			"sponsors":  sponsors,
			"total":     len(sponsors),
			"timestamp": time.Now(),
		}

		cachedData = data
		cacheTime = time.Now()
		sponsorsManager.Broadcast(data)
	}
}

func startSpecificWorkshopBroadcaster(workshopID string, manager *ClientManager) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var cachedResponse models.WorkshopScheduleResponse
	var cacheTime time.Time
	cacheDuration := 2 * time.Second

	for range ticker.C {
		if manager.Count() == 0 {
			// Kimse yoksa cleanup
			workshopSchedLock.Lock()
			delete(workshopSchedManagers, workshopID)
			workshopSchedLock.Unlock()
			manager.Shutdown()
			return
		}

		if time.Since(cacheTime) < cacheDuration {
			manager.Broadcast(cachedResponse)
			continue
		}

		var workshop models.Workshops
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := in.DB.WithContext(ctx).
			Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
				return db.Order("slot_order ASC")
			}).
			Preload("TimeSlots.Faciliator").
			First(&workshop, workshopID).Error
		cancel()

		if err != nil {
			manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
			continue
		}

		now := time.Now()
		var currentSlot *models.TimeSlotResponse
		allSlots := make([]models.TimeSlotResponse, 0, len(workshop.TimeSlots))

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

		cachedResponse = response
		cacheTime = now
		manager.Broadcast(response)
	}
}

func startWorkshopCurrentSlotBroadcaster(workshopID string, manager *ClientManager) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var cachedResponse gin.H
	var cacheTime time.Time
	cacheDuration := 1 * time.Second

	for range ticker.C {
		if manager.Count() == 0 {
			workshopCurrentSlotLock.Lock()
			delete(workshopCurrentSlotManagers, workshopID)
			workshopCurrentSlotLock.Unlock()
			manager.Shutdown()
			return
		}

		if time.Since(cacheTime) < cacheDuration && cachedResponse != nil {
			manager.Broadcast(cachedResponse)
			continue
		}

		now := time.Now()
		var workshop models.Workshops

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := in.DB.WithContext(ctx).
			Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
				return db.Order("slot_order ASC")
			}).
			Preload("TimeSlots.Faciliator").
			First(&workshop, workshopID).Error
		cancel()

		if err != nil {
			manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
			continue
		}

		var currentSlot *models.TimeSlotResponse
		var nextSlot *models.TimeSlotResponse

		for i, slot := range workshop.TimeSlots {
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
		}

		cachedResponse = response
		cacheTime = now
		manager.Broadcast(response)
	}
}
