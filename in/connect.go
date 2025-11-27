package in

import (
	"os"
	"time"

	log "github.com/jeanphorn/log4go"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	log.LoadConfiguration("./log4go.json")

	err := godotenv.Load("C:/Users/poizd/Desktop/gdgbursatestleri/devtv/in/devtv.env")
	if err != nil {
		log.Warn(".env dosyası yüklenemedi veya bulunamadı: ", err)
	}

	dsn := os.Getenv("dsn")

	if dsn == "" {
		log.Critical("DSN ortam değişkeni bulunamadı! Lütfen .env dosyasını veya ortam değişkenlerini kontrol edin.")
		return
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Error("Veritabanı bağlantısı başarısız oldu: ", err)
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Error("sql.DB alınamadı. ", err)
	}

	sqlDB.SetMaxIdleConns(15)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	log.Fine("Veritabanına başarıyla bağlanıldı.")
	log.Info("Connection Pooling ayarları: Max Idle Con: 15, Max Open Con: 50, Max Con Lifetime: 5min, Max Con Idle Time: 1min")
}
