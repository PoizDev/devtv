package main

import (
	"context"
	"devtv/controllers"
	"devtv/in"
	middlawares "devtv/middlewares"
	middlewares "devtv/middlewares"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
)

func initialize() {
	in.Connect()
	in.AutoMigrate()
	log.LoadConfiguration("./log4go.json")
}

func main() {
	initialize()
	r := gin.Default()
	r.Use(cors.Default())

	middlewares.StartHealthCollector()
	circuitBreaker := middlewares.NewCircuitBreaker(
		15, // 15 hata sonra aç
		30*time.Second,
	)
	r.GET("/circuitbreaker", func(c *gin.Context) {
		state := circuitBreaker.GetState()
		failures := circuitBreaker.GetFailures()

		stateText := "CLOSED"
		switch state {
		case 1: // StateOpen
			stateText = "OPEN"
		case 2: // StateHalfOpen
			stateText = "HALF-OPEN"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"circuit_breaker": stateText,
			"failures":        failures,
		})
	})

	r.Use(middlewares.CircuitBreakerMiddleware(circuitBreaker))
	r.Use(middlewares.TimeoutMiddleware(5 * time.Minute))

	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)

	r.GET("/faciliator", controllers.GetAllFaciliators)

	r.GET("/sponsors", controllers.GetSponsors)

	r.GET("/ws/current", controllers.GetCurrentSlotsWS)
	r.GET("/ws/workshop/:id/schedule", controllers.GetWorkshopScheduleWS)
	r.GET("/ws/upcoming", controllers.GetUpcomingSlotsWS)
	r.GET("/ws/sponsors", controllers.GetSponsorsWS)
	r.GET("/health", func(c *gin.Context) {
		health := middlewares.GetCachedHealthData()
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"data":      health,
		})
	})

	r.GET("/workshops", controllers.GetAllWorkshops)
	r.GET("/workshops/:id/schedule", controllers.GetWorkshopSchedule)
	r.GET("/workshops/current", controllers.GetCurrentSlots)
	r.GET("/workshops/upcoming", controllers.GetUpcomingSlots)

	admin := r.Group("/admin")
	admin.Use(middlawares.AuthMiddleware())
	{
		admin.GET("/users", controllers.GetAllUsers)

		admin.POST("/create/faciliator", controllers.CreateFaciliator)
		admin.PUT("/faciliator/:id", controllers.UpdateFaciliator)

		admin.DELETE("/users/:id")

		admin.POST("sponsors/add", controllers.CreateSponsor)
		admin.DELETE("sponsors/id", controllers.DeleteSponsors)

		admin.DELETE("faciliator/:id", controllers.DeleteFacilitator)
		admin.POST("/create/sponsor", controllers.CreateSponsor)
		admin.POST("/workshops/create", controllers.CreateWorkshopWithSlots)
		admin.POST("/workshops/:id/slots", controllers.AddSlotsToWorkshop)
		admin.PUT("/workshops/:id/delay", controllers.AddDelayToWorkshop)
		admin.PUT("/workshops/:id/live", controllers.SetWorkshopLive)
		admin.PUT("/workshops/:id", controllers.UpdateWorkshops)
		admin.DELETE("/workshops/:id", controllers.DeleteWorkshop)

		admin.DELETE("/slots/:id", controllers.DeleteSlots)
		admin.PUT("/slots/:id", controllers.UpdateTimeSlot)
	}

	srv := &http.Server{
		Addr:    ":2012",
		Handler: r,
	}

	go func() {
		log.Info("Server Başlatılıyor - Port, ", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Critical("Server başlatılamadı")
		}
	}()

	//shutdown sinyalini dinlemesi için
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Critical("Server zorla kapatıldı, ", err.Error())
	}

	sqlDB, _ := in.DB.DB()
	if err := sqlDB.Close(); err != nil {
		log.Error("DB Bağlantısı kapanamadı: %s", err)
	}

	log.Info("Server kapatıldı.")
}

/* Cors planlaması:

	AllowOrigins:     []string{
	"https://devfestbursa.com"
	"https://www.devfestbursa.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
    ExposeHeaders: []string{"Content-Length", "Set-Cookie"}
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,*/
