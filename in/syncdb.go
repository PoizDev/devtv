package in

import "devtv/models"

func AutoMigrate() {
	if DB == nil {
		Connect()
	}
	DB.AutoMigrate(
		&models.Faciliators{},
		&models.User{},
		&models.Workshops{},
		&models.WorkshopTimeSlot{},
	)
}
