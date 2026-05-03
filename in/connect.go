package in

import (
	"devtv/config" // Config paketini import et
	"os"

	log "github.com/jeanphorn/log4go"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect artık config verilerini parametre olarak alıyor
func Connect(dbConf config.DatabaseConfig, envPath string) {
	// Config'den gelen env yolunu kullanıyoruz
	err := godotenv.Load(envPath)
	if err != nil {
		log.Warn(".env dosyası yüklenemedi (%s): %v", envPath, err)
		return
	}

	dsn := os.Getenv("dsn")
	if dsn == "" {
		log.Critical("DSN ortam değişkeni bulunamadı! Lütfen .env dosyasını kontrol edin.")
		// Kritik hata olduğu için burada panic atmak veya os.Exit yapmak düşünülebilir
		// ancak çağıran yerin (main) akışı yönetmesine izin veriyoruz.
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
		return
	}

	// Config dosyasından gelen değerleri set ediyoruz
	sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbConf.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConf.ConnMaxIdleTime)

	log.Fine("Veritabanına başarıyla bağlanıldı.")
	log.Info("Connection Pooling ayarları uygulandı.")
}
