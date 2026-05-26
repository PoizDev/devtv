package in

import (
	"devtv/config"
	"devtv/models"
	"os"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func AutoMigrate() {
	if DB == nil {
		return
	}

	//' Eski typo düzeltme migration'ları — faciliators → facilitators
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
		&models.Category{},
		&models.Tag{},
		&models.SurveyQuestion{},
		&models.SurveyOption{},
		&models.UserSurveyResponse{},
	)
}

func SeedAdminUser() {
	if DB == nil {
		return
	}

	var count int64
	DB.Model(&models.User{}).Count(&count)

	if count > 0 {
		config.Log.Info("Veritabanında kullanıcı mevcut, seed işlemi atlandı.")
		return
	}

	config.Log.Info("Veritabanında hiç kullanıcı bulunamadığından varsayılan admin oluşturuluyor.")

	adminUser := os.Getenv("DEFAULT_ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("DEFAULT_ADMIN_PASS")
	if adminPass == "" {
		config.Log.Warn("DEFAULT_ADMIN_PASS .env dosyasından okunamadı, admin oluşturma atlanıyor")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), 10)
	if err != nil {
		config.Log.Fatal("Varsayılan admin şifresi hashlenirken hata oluştu", zap.Error(err))
	}

	admin := models.User{
		Username: adminUser,
		Password: string(hash),
		Role:     "admin",
	}

	if result := DB.Create(&admin); result.Error != nil {
		config.Log.Fatal("Varsayılan admin oluşturulurken hata oluştu", zap.Error(result.Error))
	}

	config.Log.Info("Varsayılan admin oluşturuldu", zap.String("username", adminUser))
}
