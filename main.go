package main

import (
	"devtv/controllers"
	"devtv/in"
	middlawares "devtv/middlewares"
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

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Use(middlawares.TimeoutMiddleware(5 * time.Minute))
	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)

	r.GET("/faciliator", controllers.GetAllFaciliators)

	r.GET("/workshops", controllers.GetAllWorkshops)
	r.GET("/workshops/:id/schedule", controllers.GetWorkshopSchedule)
	r.GET("/workshops/current", controllers.GetCurrentSlot)
	r.GET("/workshops/upcoming", controllers.GetUpcomingSlots)

	admin := r.Group("/admin")
	admin.Use(middlawares.AuthMiddleware())
	{
		admin.GET("/users", controllers.GetAllUsers)

		admin.POST("/create/faciliator", controllers.CreateFaciliator)

		admin.POST("/create/sponsor", controllers.CreateSponsor)
		admin.POST("/workshops/create", controllers.CreateWorkshopWithSlots)
		admin.POST("/workshops/:id/slots", controllers.AddSlotsToWorkshop)
		admin.PUT("/workshops/:id/delay", controllers.AddDelayToWorkshop)
		admin.PUT("/workshops/:id/live", controllers.SetWorkshopLive)
	}

	log.Info("Server başlatılıyor - Port: 2012")
	r.Run(":2012")
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
