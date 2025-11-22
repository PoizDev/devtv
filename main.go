package main

import (
	"devtv/controllers"
	"devtv/in"

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

	r.GET("/users", controllers.GetAllUsers)
	r.GET("/faciliator", controllers.GetAllFaciliators)

	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)
	r.POST("/create/faciliator", controllers.CreateFaciliator)

	r.Run(":2012")
}
