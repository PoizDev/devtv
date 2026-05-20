package in

import (
	"devtv/models"

	log "github.com/jeanphorn/log4go"
)

func AutoMigrate() {
	if DB != nil {
		//' Faciliator → Facilitator typo düzeltmesi: tablo ve kolon rename
		if DB.Migrator().HasTable("faciliators") {
			log.Info("faciliators tablosu facilitators olarak yeniden adlandırılıyor...")
			if err := DB.Exec("ALTER TABLE faciliators RENAME TO facilitators").Error; err != nil {
				log.Error("Tablo yeniden adlandırılamadı: ", err)
			}
		}
		if DB.Migrator().HasTable("facilitators") && DB.Migrator().HasColumn(&models.Facilitators{}, "faciliator_id") {
			log.Info("faciliator_id kolonu facilitator_id olarak yeniden adlandırılıyor...")
			if err := DB.Exec("ALTER TABLE facilitators RENAME COLUMN faciliator_id TO facilitator_id").Error; err != nil {
				log.Error("Kolon yeniden adlandırılamadı: ", err)
			}
		}
		if DB.Migrator().HasTable("workshop_time_slots") {
			if DB.Migrator().HasColumn(&models.WorkshopTimeSlot{}, "faciliator_id") {
				log.Info("workshop_time_slots.faciliator_id kolonu facilitator_id olarak yeniden adlandırılıyor...")
				if err := DB.Exec("ALTER TABLE workshop_time_slots RENAME COLUMN faciliator_id TO facilitator_id").Error; err != nil {
					log.Error("workshop_time_slots kolon yeniden adlandırılamadı: ", err)
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
