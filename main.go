package main

import (
	"context"
	"devtv/config" // Config paketi
	"devtv/controllers"
	"devtv/in"
	"devtv/middlewares"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
	"golang.org/x/time/rate"
)

var conf *config.Config

func initialize() {
	var err error
	//'İçeriye aktaracaklarım burada

	conf, err = config.LoadConfig("conf.yaml")
	if err != nil {
		panic("Config dosyası (conf.yaml) yüklenemedi: " + err.Error())
	}

	log.LoadConfiguration(conf.Server.LogConfigPath)

	in.Connect(conf.Database, conf.Server.EnvPath)

	in.AutoMigrate()
}

func main() {
	initialize()

	r := gin.Default()
	r.Use(cors.Default())

	middlewares.StartHealthCollector()

	//'Sağlık için kontrolcüler setuplar falan filan
	circuitBreaker := middlewares.NewCircuitBreaker(
		int(conf.Middleware.CircuitBreaker.Threshold), //'Conf'dan alıyor artık her şeyi
		conf.Middleware.CircuitBreaker.Timeout,
	)
	//'Circuit Breaker dalgası
	r.GET("/circuitbreaker", func(c *gin.Context) {
		state := circuitBreaker.GetState()
		failures := circuitBreaker.GetFailures()
		stateText := "CLOSED"
		switch state {
		case 1:
			stateText = "OPEN"
		case 2:
			stateText = "HALF-OPEN"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"circuit_breaker": stateText,
			"failures":        failures,
		})
	})

	//'IP Limiter tayfası DDoS engellemek için bu
	rateLimiter := middlewares.NewIPRateLimiter(
		rate.Limit(conf.Middleware.RateLimit.Limit),
		conf.Middleware.RateLimit.Burst,
	)
	//'Middleware Tayfayı burada kullandırıyorum
	r.Use(middlewares.MetricsMiddleware())
	r.Use(middlewares.RateLimitMiddleware(rateLimiter))
	r.Use(middlewares.CircuitBreakerMiddleware(circuitBreaker))
	r.Use(middlewares.TimeoutMiddleware(conf.Middleware.RequestTimeout))
	r.Use(middlewares.RequestLoggerMiddleWare())

	//'Auth tayfası
	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)
	//'Konuşmacılar / Atölye tayfa
	r.GET("/faciliator", controllers.GetAllFaciliators)
	//'Sponsorluk görüntüleme
	r.GET("/sponsors", controllers.GetSponsors)
	//'WebSocketler
	r.GET("/ws/current", controllers.GetCurrentSlotsWS)
	r.GET("/ws/:id/current", controllers.GetCurrentSlotInWorkshopWS)
	r.GET("/ws/workshop/:id/schedule", controllers.GetWorkshopScheduleWS)
	r.GET("/ws/upcoming", controllers.GetUpcomingSlotsWS)
	r.GET("/ws/sponsors", controllers.GetSponsorsWS)

	//'Workshop HTTP istekleri
	r.GET("/workshops", controllers.GetAllWorkshops)
	r.GET("/workshops/:id/schedule", controllers.GetWorkshopSchedule)
	r.GET("/workshops/current", controllers.GetCurrentSlots)
	r.GET("/workshops/upcoming", controllers.GetUpcomingSlots)
	r.GET("/workshop/:id/slots", controllers.GetCurrentSlotInWorkshop)

	r.GET("/health", func(c *gin.Context) {
		health := middlewares.GetCachedHealthData()
		metrics := middlewares.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"data":      health,
			"metrics":   metrics,
		})
	})

	r.LoadHTMLGlob("static/admin/*")

	r.GET("/panel", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"Title": "DevTV Kontrol Paneli",
		})
	})

	//'Admin Accessi
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.GET("/users", controllers.GetAllUsers)
		admin.DELETE("/users/:id", controllers.DeleteUser)
		admin.PUT("/users/:id", controllers.UpdateUser)

		admin.POST("/create/faciliator", controllers.CreateFaciliator)
		admin.PUT("/faciliator/:id", controllers.UpdateFaciliator)
		admin.DELETE("faciliator/:id", controllers.DeleteFacilitator)

		admin.POST("sponsors/add", controllers.CreateSponsor)
		admin.DELETE("sponsors/id", controllers.DeleteSponsors)
		admin.POST("/create/sponsor", controllers.CreateSponsor)

		admin.POST("/workshops/create", controllers.CreateWorkshopWithSlots)
		admin.POST("/workshops/:id/slots", controllers.AddSlotsToWorkshop)
		admin.PUT("/workshops/:id/delay", controllers.AddDelayToWorkshop)
		admin.PUT("/workshops/:id", controllers.UpdateWorkshops)
		admin.DELETE("/workshops/:id", controllers.DeleteWorkshop)

		admin.DELETE("/slots/:id", controllers.DeleteSlots)
		admin.PUT("/slots/:id", controllers.UpdateTimeSlot)
	}

	// 'Server Config ayarları
	srv := &http.Server{
		Addr:    conf.Server.Port,
		Handler: r,
	}

	go func() {
		log.Info("Server Başlatılıyor - Port: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Critical("Server başlatılamadı: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	//'Shutdown Timeout
	ctx, cancel := context.WithTimeout(context.Background(), conf.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Critical("Server zorla kapatıldı: %s", err.Error())
	}

	sqlDB, _ := in.DB.DB()
	if err := sqlDB.Close(); err != nil {
		log.Error("DB Bağlantısı kapanamadı: %s", err)
	}

	log.Info("Server kapatıldı.")
}

/* //' Cors planlaması, live'a alınırken bu kullanılacak:

	AllowOrigins:     []string{
	"https://devfestbursa.com"
	"https://www.devfestbursa.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
    ExposeHeaders: []string{"Content-Length", "Set-Cookie"}
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,*/
