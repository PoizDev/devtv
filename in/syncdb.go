package in

import "devtv/models"

func AutoMigrate() {
	if DB != nil {
		DB.AutoMigrate(
			&models.Faciliators{},
			&models.User{},
			&models.Workshops{},
			&models.WorkshopTimeSlot{},
			&models.Sponsors{},
		)
	}
}
