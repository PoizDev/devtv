package in

import (
	"os"

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
	log.Fine("Veritabanına başarıyla bağlanıldı.")
}
