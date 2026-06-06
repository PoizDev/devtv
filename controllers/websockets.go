package controllers

import (
	"context"
	"devtv/config"
	"devtv/in"
	"devtv/middlewares"
	"devtv/models"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

var wsAllowedOrigins map[string]bool
var wsSingleflight singleflight.Group

func SetWSAllowedOrigins(origins []string) {
	wsAllowedOrigins = make(map[string]bool, len(origins))
	for _, o := range origins {
		wsAllowedOrigins[o] = true
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true //' Same-origin isteklerde Origin header'ı olmayabilir
		}
		return wsAllowedOrigins[origin]
	},
	HandshakeTimeout: 10 * time.Second,
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			c.conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type ClientManager struct {
	clients      map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	broadcast    chan interface{}
	broadcastRaw chan []byte
	lock         sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewClientManager() *ClientManager {
	ctx, cancel := context.WithCancel(context.Background())
	cm := &ClientManager{
		clients:      make(map[*Client]bool),
		register:     make(chan *Client, 100),
		unregister:   make(chan *Client, 100),
		broadcast:    make(chan interface{}, 1000),
		broadcastRaw: make(chan []byte, 1000),
		ctx:          ctx,
		cancel:       cancel,
	}
	go cm.run()
	return cm
}

func (cm *ClientManager) run() {
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
				close(client.send)
			}
			cm.lock.Unlock()

		case raw := <-cm.broadcastRaw:
			cm.lock.Lock()
			for client := range cm.clients {
				select {
				case client.send <- raw:
				default:
					close(client.send)
					delete(cm.clients, client)
				}
			}
			cm.lock.Unlock()

		case message := <-cm.broadcast:
			raw, err := json.Marshal(message)
			if err != nil {
				continue
			}
			cm.lock.Lock()
			for client := range cm.clients {
				select {
				case client.send <- raw:
				default:
					close(client.send)
					delete(cm.clients, client)
				}
			}
			cm.lock.Unlock()
		}
	}
}

func (cm *ClientManager) Add(client *Client) {
	cm.register <- client
}

func (cm *ClientManager) Remove(client *Client) {
	cm.unregister <- client
}

func (cm *ClientManager) BroadcastRaw(data []byte) {
	select {
	case cm.broadcastRaw <- data:
	default:
		config.Log.Warn("BroadcastRaw buffer full, dropping message")
	}
}

func (cm *ClientManager) Broadcast(message interface{}) {
	select {
	case cm.broadcast <- message:
	default:
		config.Log.Warn("Broadcast buffer full, dropping message")
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

func marshalOrLog(v interface{}) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		config.Log.Error("JSON marshal error", zap.Error(err))
		return nil
	}
	return raw
}

// ' DB hatası durumunda Redis → RAM fallback zinciri
func tryFallback(manager *ClientManager, cachedJSON []byte, redisFallbackKey string) {
	if in.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		redisData, err := in.RDB.Get(ctx, redisFallbackKey).Bytes()
		cancel()

		if err == nil && len(redisData) > 0 {
			config.Log.Warn("Redis fallback aktif", zap.String("key", redisFallbackKey))
			manager.BroadcastRaw(redisData)
			return
		}
	}

	if cachedJSON != nil {
		config.Log.Error("Redis de yok, RAM cache kullanılıyor")
		manager.BroadcastRaw(cachedJSON)
	}
}

func cacheAndBroadcast(manager *ClientManager, raw []byte, cache *[]byte, cacheTime *time.Time, redisFallbackKey string) {
	*cache = raw
	*cacheTime = time.Now()

	if in.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err := in.RDB.Set(ctx, redisFallbackKey, raw, 1*time.Hour).Err(); err != nil {
			config.Log.Warn("Redis yedeklemesi başarısız", zap.Error(err))
		}
		cancel()
	}

	manager.BroadcastRaw(raw)
}

// ' DB sorgularını singleflight ile sararak eşzamanlı istekleri tekile indirger
func fetchCurrentSlotsData() ([]models.WorkshopTimeSlot, error) {
	val, err, _ := wsSingleflight.Do("ws:current_slots", func() (interface{}, error) {
		now := time.Now()
		var slots []models.WorkshopTimeSlot
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := in.DB.WithContext(ctx).
			Preload("Facilitator").
			Preload("Workshop").
			Where("slot_start <= ? AND slot_end >= ?", now, now).
			Find(&slots).Error
		if err != nil {
			return nil, err
		}
		return slots, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.WorkshopTimeSlot), nil
}

func fetchUpcomingSlotsData() ([]models.WorkshopTimeSlot, error) {
	val, err, _ := wsSingleflight.Do("ws:upcoming_slots", func() (interface{}, error) {
		now := time.Now()
		var slots []models.WorkshopTimeSlot
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := in.DB.WithContext(ctx).
			Preload("Facilitator").
			Preload("Workshop").
			Where("slot_start > ?", now).
			Order("slot_start ASC").
			Find(&slots).Error
		if err != nil {
			return nil, err
		}
		return slots, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.WorkshopTimeSlot), nil
}

func fetchSponsorsData() ([]models.Sponsors, error) {
	val, err, _ := wsSingleflight.Do("ws:sponsors", func() (interface{}, error) {
		var sponsors []models.Sponsors
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := in.DB.WithContext(ctx).Find(&sponsors).Error
		if err != nil {
			return nil, err
		}
		return sponsors, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.Sponsors), nil
}

// ' Schedule ve current-slot broadcaster'ları aynı workshop verisini paylaşır
func fetchWorkshopWithSlots(workshopID string) (*models.Workshops, error) {
	val, err, _ := wsSingleflight.Do("ws:workshop:"+workshopID, func() (interface{}, error) {
		var workshop models.Workshops
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := in.DB.WithContext(ctx).
			Preload("TimeSlots", func(db *gorm.DB) *gorm.DB {
				return db.Order("slot_order ASC")
			}).
			Preload("TimeSlots.Facilitator").
			First(&workshop, workshopID).Error
		if err != nil {
			return nil, err
		}
		return &workshop, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.Workshops), nil
}

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

		go startCurrentSlotsBroadcaster()
		go startUpcomingSlotsBroadcaster()
		go startSponsorsBroadcaster()
	})
}

func setupWSClient(c *gin.Context, manager *ClientManager) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		config.Log.Error("WS upgrade error", zap.Error(err))
		return
	}
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	middlewares.IncreaseWS()

	client := &Client{
		conn: ws,
		send: make(chan []byte, 256),
	}
	manager.Add(client)
	go client.writePump()

	go func() {
		defer middlewares.DecreaseWS()
		defer manager.Remove(client)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func GetCurrentSlotsWS(c *gin.Context) {
	setupWSClient(c, currentSlotsManager)
}

func GetUpcomingSlotsWS(c *gin.Context) {
	setupWSClient(c, upcomingSlotsManager)
}

func GetSponsorsWS(c *gin.Context) {
	setupWSClient(c, sponsorsManager)
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
	workshopSchedLock.Lock()
	manager, exists := workshopSchedManagers[workshopID]
	if !exists {
		manager = NewClientManager()
		workshopSchedManagers[workshopID] = manager
		go startSpecificWorkshopBroadcaster(workshopID, manager)
	}
	workshopSchedLock.Unlock()
	middlewares.IncreaseWS()

	client := &Client{
		conn: ws,
		send: make(chan []byte, 256),
	}
	manager.Add(client)
	go client.writePump()

	go func() {
		defer middlewares.DecreaseWS()
		defer manager.Remove(client)
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
		config.Log.Error("WS upgrade error", zap.Error(err))
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
	middlewares.IncreaseWS()

	client := &Client{
		conn: ws,
		send: make(chan []byte, 256),
	}
	manager.Add(client)
	go client.writePump()

	go func() {
		defer middlewares.DecreaseWS()
		defer manager.Remove(client)
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func startCurrentSlotsBroadcaster() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var cachedJSON []byte
	var cacheTime time.Time
	cacheDuration := 3 * time.Second
	redisFallbackKey := "devtv:ws_fallback:current_slots"

	for range ticker.C {
		if currentSlotsManager.Count() == 0 {
			continue
		}

		if cachedJSON != nil && time.Since(cacheTime) < cacheDuration {
			currentSlotsManager.BroadcastRaw(cachedJSON)
			continue
		}

		now := time.Now()
		slots, err := fetchCurrentSlotsData()
		if err != nil {
			config.Log.Error("Veritabanı hatası, fallback devreye giriyor", zap.Error(err))
			tryFallback(currentSlotsManager, cachedJSON, redisFallbackKey)
			continue
		}

		var data gin.H
		if len(slots) == 0 {
			data = gin.H{
				"message":   "Şu anda aktif slot yok",
				"slots":     []models.TimeSlotResponse{},
				"total":     0,
				"timestamp": now,
			}
		} else {
			workshopInfo := make([]gin.H, 0, len(slots))
			for _, slot := range slots {
				workshopInfo = append(workshopInfo, gin.H{
					"workshop_id":   slot.Workshop.WorkshopID,
					"workshop_name": slot.Workshop.WorkshopName,
					"slot":          slot.ToResponse(),
				})
			}
			data = gin.H{
				"active_workshops": workshopInfo,
				"total":            len(workshopInfo),
				"timestamp":        now,
			}
		}

		raw := marshalOrLog(data)
		if raw == nil {
			continue
		}

		cacheAndBroadcast(currentSlotsManager, raw, &cachedJSON, &cacheTime, redisFallbackKey)
	}
}

func startUpcomingSlotsBroadcaster() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var cachedJSON []byte
	var cacheTime time.Time
	cacheDuration := 4 * time.Second
	redisFallbackKey := "devtv:ws_fallback:upcoming_slots"

	for range ticker.C {
		if upcomingSlotsManager.Count() == 0 {
			continue
		}

		if cachedJSON != nil && time.Since(cacheTime) < cacheDuration {
			upcomingSlotsManager.BroadcastRaw(cachedJSON)
			continue
		}

		now := time.Now()
		slots, err := fetchUpcomingSlotsData()
		if err != nil {
			config.Log.Error("Veritabanı hatası, fallback devreye giriyor", zap.Error(err))
			tryFallback(upcomingSlotsManager, cachedJSON, redisFallbackKey)
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
				response = append(response, models.UpcomingSlotResponse{
					SlotID:         slot.SlotID,
					WorkshopName:   slot.Workshop.WorkshopName,
					SlotStart:      slot.SlotStart,
					SlotEnd:        slot.SlotEnd,
					Facilitator:    slot.Facilitator.ToResponse(),
					TimeUntilStart: formatDuration(slot.SlotStart.Sub(now)),
				})
			}
			data = gin.H{
				"upcoming_slots": response,
				"total":          len(response),
				"timestamp":      now,
			}
		}

		raw := marshalOrLog(data)
		if raw == nil {
			continue
		}

		cacheAndBroadcast(upcomingSlotsManager, raw, &cachedJSON, &cacheTime, redisFallbackKey)
	}
}

func startSponsorsBroadcaster() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var cachedJSON []byte
	var cacheTime time.Time
	cacheDuration := 15 * time.Second
	redisFallbackKey := "devtv:ws_fallback:sponsors"

	for range ticker.C {
		if sponsorsManager.Count() == 0 {
			continue
		}

		if cachedJSON != nil && time.Since(cacheTime) < cacheDuration {
			sponsorsManager.BroadcastRaw(cachedJSON)
			continue
		}

		sponsors, err := fetchSponsorsData()
		if err != nil {
			config.Log.Error("Veritabanı hatası, fallback devreye giriyor", zap.Error(err))
			tryFallback(sponsorsManager, cachedJSON, redisFallbackKey)
			continue
		}

		data := gin.H{
			"sponsors":  sponsors,
			"total":     len(sponsors),
			"timestamp": time.Now(),
		}

		raw := marshalOrLog(data)
		if raw == nil {
			continue
		}

		cacheAndBroadcast(sponsorsManager, raw, &cachedJSON, &cacheTime, redisFallbackKey)
	}
}

func startSpecificWorkshopBroadcaster(workshopID string, manager *ClientManager) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var cachedJSON []byte
	var cacheTime time.Time
	cacheDuration := 4 * time.Second
	redisFallbackKey := "devtv:ws_fallback:workshop_schedule:" + workshopID

	for range ticker.C {
		if manager.Count() == 0 {
			workshopSchedLock.Lock()
			delete(workshopSchedManagers, workshopID)
			workshopSchedLock.Unlock()
			manager.Shutdown()
			return
		}

		if cachedJSON != nil && time.Since(cacheTime) < cacheDuration {
			manager.BroadcastRaw(cachedJSON)
			continue
		}

		workshop, err := fetchWorkshopWithSlots(workshopID)
		if err != nil {
			config.Log.Error("Veritabanı hatası, fallback devreye giriyor", zap.Error(err))
			tryFallback(manager, cachedJSON, redisFallbackKey)
			if cachedJSON == nil {
				manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
			}
			continue
		}

		now := time.Now()
		var currentSlot *models.TimeSlotResponse
		allSlots := make([]models.TimeSlotResponse, 0, len(workshop.TimeSlots))

		for _, slot := range workshop.TimeSlots {
			slotResponse := slot.ToResponse()
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

		raw := marshalOrLog(response)
		if raw == nil {
			continue
		}

		cacheAndBroadcast(manager, raw, &cachedJSON, &cacheTime, redisFallbackKey)
	}
}

func startWorkshopCurrentSlotBroadcaster(workshopID string, manager *ClientManager) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	redisFallbackKey := "devtv:ws_fallback:workshop_current_slot:" + workshopID

	var cachedJSON []byte
	var cacheTime time.Time
	cacheDuration := 3 * time.Second

	for range ticker.C {
		if manager.Count() == 0 {
			workshopCurrentSlotLock.Lock()
			delete(workshopCurrentSlotManagers, workshopID)
			workshopCurrentSlotLock.Unlock()
			manager.Shutdown()
			return
		}

		if cachedJSON != nil && time.Since(cacheTime) < cacheDuration {
			manager.BroadcastRaw(cachedJSON)
			continue
		}

		now := time.Now()
		workshop, err := fetchWorkshopWithSlots(workshopID)
		if err != nil {
			config.Log.Error("Veritabanı hatası, fallback devreye giriyor", zap.Error(err))
			tryFallback(manager, cachedJSON, redisFallbackKey)
			if cachedJSON == nil {
				manager.Broadcast(gin.H{"error": "Workshop bulunamadı"})
			}
			continue
		}

		var currentSlot *models.TimeSlotResponse
		var nextSlot *models.TimeSlotResponse

		for i, slot := range workshop.TimeSlots {
			if now.After(slot.SlotStart) && now.Before(slot.SlotEnd) {
				resp := slot.ToResponse()
				currentSlot = &resp
				if i+1 < len(workshop.TimeSlots) {
					nxtResp := workshop.TimeSlots[i+1].ToResponse()
					nextSlot = &nxtResp
				}
				break
			}
		}

		if currentSlot == nil {
			for _, slot := range workshop.TimeSlots {
				if now.Before(slot.SlotStart) {
					resp := slot.ToResponse()
					nextSlot = &resp
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
			response["next_slot"] = nextSlot
			response["time_until_next"] = formatDuration(nextSlot.SlotStart.Sub(now))
		} else {
			response["next_slot"] = nil
		}

		raw := marshalOrLog(response)
		if raw == nil {
			continue
		}

		cacheAndBroadcast(manager, raw, &cachedJSON, &cacheTime, redisFallbackKey)
	}
}
