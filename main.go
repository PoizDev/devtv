package main

import (
	"context"
	"devtv/config"
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
	"go.uber.org/zap"
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

	config.InitLogger(conf.Server.ActiveLevel)

	in.Connect(conf.Database, conf.Redis, conf.Auth, conf.Server.EnvPath)

	in.AutoMigrate()
	in.SeedAdminUser()
}

func main() {
	initialize()
	defer func() {
		if config.Log != nil {
			_ = config.Log.Sync()
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())

	//'Health istekleri loglanmaz
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{
			"/health",
			"/health/check",
		},
	}))

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:5500",
		"http://localhost:5500", // Bunu ekleyin
		"http://127.0.0.1:5500/frontend/index.html",
		"http://localhost", // Bunu da ekleyin
		"http://127.0.0.1", // Bunu da
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowWebSockets = true
	r.Use(cors.New(corsConfig))

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

	cachedRoutes := r.Group("/")
	cachedRoutes.Use(middlewares.RedisFallbackCache(in.RDB, 5*time.Second))
	{
		//'Konuşmacılar / Atölye tayfa
		cachedRoutes.GET("/facilitator", controllers.GetAllFacilitators)
		//'Sponsorluk görüntüleme
		cachedRoutes.GET("/sponsors", controllers.GetSponsors)
		cachedRoutes.GET("/workshops", controllers.GetAllWorkshops)
		cachedRoutes.GET("/workshops/:id/schedule", controllers.GetWorkshopSchedule)
		cachedRoutes.GET("/workshops/current", controllers.GetCurrentSlots)
		cachedRoutes.GET("/workshops/upcoming", controllers.GetUpcomingSlots)
		cachedRoutes.GET("/workshop/:id/slots", controllers.GetCurrentSlotInWorkshop)
	}

	//'Auth tayfası
	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)
	//'WebSocketler
	r.GET("/ws/current", controllers.GetCurrentSlotsWS)
	r.GET("/ws/:id/current", controllers.GetCurrentSlotInWorkshopWS)
	r.GET("/ws/workshop/:id/schedule", controllers.GetWorkshopScheduleWS)
	r.GET("/ws/upcoming", controllers.GetUpcomingSlotsWS)
	r.GET("/ws/sponsors", controllers.GetSponsorsWS)

	//'Survey tayfası (Sadece yetkili kullanıcılar anket çözebilir)
	survey := r.Group("/survey")
	survey.Use(middlewares.AuthMiddleware())
	{
		survey.GET("/questions", controllers.GetActiveQuestions)
		survey.POST("/submit", controllers.SubmitSurvey)
		survey.GET("/results", controllers.GetSurveyResults)
	}

	//'Workshop HTTP istekleri

	//' Protobuf health endpoint'leri — daha küçük payload, daha hızlı serialize
	r.GET("/health", middlewares.ProtoHealthHandler)
	r.GET("/health/check", middlewares.ProtoHealthCheckHandler)

	//'Admin Accessi
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware(), middlewares.AdminMiddleware())
	{
		admin.GET("/users", controllers.GetAllUsers)
		admin.DELETE("/users/:id", controllers.DeleteUser)
		admin.PUT("/users/:id", controllers.UpdateUser)

		admin.POST("/create/facilitator", controllers.CreateFacilitator)
		admin.PUT("/facilitator/:id", controllers.UpdateFacilitator)
		admin.DELETE("facilitator/:id", controllers.DeleteFacilitator)

		admin.POST("sponsors/add", controllers.CreateSponsor)
		admin.DELETE("sponsors/:id", controllers.DeleteSponsors)
		admin.POST("/create/sponsors", controllers.CreateSponsor)
		admin.PUT("/sponsors/:id", controllers.UpdateSponsor)

		admin.POST("/workshops/create", controllers.CreateWorkshopWithSlots)
		admin.POST("/workshops/:id/slots", controllers.AddSlotsToWorkshop)
		admin.PUT("/workshops/:id/delay", controllers.AddDelayToWorkshop)
		admin.PUT("/workshops/:id", controllers.UpdateWorkshops)
		admin.DELETE("/workshops/:id", controllers.DeleteWorkshop)

		admin.DELETE("/slots/:id", controllers.DeleteSlots)
		admin.PUT("/slots/:id", controllers.UpdateTimeSlot)

		// Survey Admin
		admin.GET("/categories", controllers.GetAllCategories)
		admin.POST("/categories", controllers.CreateCategory)
		admin.PUT("/categories/:id", controllers.UpdateCategory)
		admin.DELETE("/categories/:id", controllers.DeleteCategory)

		admin.GET("/tags", controllers.GetAllTags)
		admin.POST("/tags", controllers.CreateTag)
		admin.PUT("/tags/:id", controllers.UpdateTag)
		admin.DELETE("/tags/:id", controllers.DeleteTag)

		admin.GET("/survey/questions", controllers.GetAllSurveyQuestions)
		admin.POST("/survey/questions", controllers.CreateSurveyQuestion)
		admin.PUT("/survey/questions/:id", controllers.UpdateSurveyQuestion)
		admin.DELETE("/survey/questions/:id", controllers.DeleteSurveyQuestion)
		
		admin.POST("/survey/options", controllers.CreateSurveyOption)
		admin.PUT("/survey/options/:id", controllers.UpdateSurveyOption)
		admin.DELETE("/survey/options/:id", controllers.DeleteSurveyOption)
	}

	// 'Server Config ayarları
	srv := &http.Server{
		Addr:    conf.Server.Port,
		Handler: r,
	}

	go func() {
		config.Log.Info("Server Başlatılıyor", zap.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Log.Fatal("Server başlatılamadı", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	//'Shutdown Timeout
	ctx, cancel := context.WithTimeout(context.Background(), conf.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		config.Log.Fatal("Server zorla kapatıldı", zap.Error(err))
	}

	sqlDB, _ := in.DB.DB()
	if err := sqlDB.Close(); err != nil {
		config.Log.Error("DB Bağlantısı kapanamadı", zap.Error(err))
	}

	config.Log.Info("Server kapatıldı.")
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
