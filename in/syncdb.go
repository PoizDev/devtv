package in

import (
	"devtv/config"
	"devtv/models"

	"go.uber.org/zap"
)

func AutoMigrate() {
	if DB != nil {
		//' Faciliator → Facilitator typo düzeltmesi: tablo ve kolon rename
		if DB.Migrator().HasTable("faciliators") {
			config.Log.Info("faciliators tablosu facilitators olarak yeniden adlandırılıyor...")
			if err := DB.Exec("ALTER TABLE faciliators RENAME TO facilitators").Error; err != nil {
				config.Log.Error("Tablo yeniden adlandırılamadı", zap.Error(err))
			}
		}
		if DB.Migrator().HasTable("facilitators") && DB.Migrator().HasColumn(&models.Facilitators{}, "faciliator_id") {
			config.Log.Info("faciliator_id kolonu facilitator_id olarak yeniden adlandırılıyor...")
			if err := DB.Exec("ALTER TABLE facilitators RENAME COLUMN faciliator_id TO facilitator_id").Error; err != nil {
				config.Log.Error("Kolon yeniden adlandırılamadı", zap.Error(err))
			}
		}
		if DB.Migrator().HasTable("workshop_time_slots") {
			if DB.Migrator().HasColumn(&models.WorkshopTimeSlot{}, "faciliator_id") {
				config.Log.Info("workshop_time_slots.faciliator_id kolonu facilitator_id olarak yeniden adlandırılıyor...")
				if err := DB.Exec("ALTER TABLE workshop_time_slots RENAME COLUMN faciliator_id TO facilitator_id").Error; err != nil {
					config.Log.Error("workshop_time_slots kolon yeniden adlandırılamadı", zap.Error(err))
				}
			}
		}

		DB.AutoMigrate(
			&models.Facilitators{},
			&models.User{},
			&models.Workshops{},
			&models.WorkshopTimeSlot{},
			&models.Sponsors{},
		)
	}
}
