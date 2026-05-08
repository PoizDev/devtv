package in

import (
	"devtv/config" // Config paketini import et
	"os"

	log "github.com/jeanphorn/log4go"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RDB *redis.Client
var Auth config.AuthConfig

func Connect(dbConf config.DatabaseConfig, redisConf config.RedisConfig, authConf config.AuthConfig, envPath string) {

	err := godotenv.Load(envPath)
	if err != nil {
		log.Warn(".env dosyası yüklenemedi (%s): %v", envPath, err)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Critical("JWT_SECRET ortam değişkeni bulunamadı! Auth sistemi çalışmayacak.")
	}
	authConf.JWTSecret = jwtSecret
	Auth = authConf
	log.Info("Auth config yüklendi — Domain: %s, Secure: %v, Token Süresi: %d gün", Auth.CookieDomain, Auth.CookieSecure, Auth.TokenExpiryDays)

	dsn := os.Getenv("dsn")
	if dsn == "" {
		log.Critical("DSN ortam değişkeni bulunamadı! Lütfen .env dosyasını kontrol edin.")
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
	RDB = redis.NewClient(&redis.Options{
		Addr:     redisConf.RedisUrl,
		Password: redisConf.RedisPwr,
		DB:       redisConf.Db,
	})

	// Config dosyasından gelen değerleri set ediyoruz
	sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbConf.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConf.ConnMaxIdleTime)

	log.Fine("Veritabanına başarıyla bağlanıldı.")
	log.Info("Connection Pooling ayarları uygulandı.")
}
