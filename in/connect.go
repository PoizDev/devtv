package in

import (
	"devtv/config" // Config paketini import et
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RDB *redis.Client
var Auth config.AuthConfig

func Connect(dbConf config.DatabaseConfig, redisConf config.RedisConfig, authConf config.AuthConfig, envPath string) {

	err := godotenv.Load(envPath)
	if err != nil {
		config.Log.Warn(".env dosyası yüklenemedi", zap.String("path", envPath), zap.Error(err))
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		config.Log.Error("JWT_SECRET ortam değişkeni bulunamadı! Auth sistemi çalışmayacak.")
	}
	authConf.JWTSecret = jwtSecret
	Auth = authConf
	config.Log.Info("Auth config yüklendi", zap.String("domain", Auth.CookieDomain), zap.Bool("secure", Auth.CookieSecure), zap.Int("expiry_days", Auth.TokenExpiryDays))

	dsn := os.Getenv("dsn")
	if dsn == "" {
		config.Log.Error("DSN ortam değişkeni bulunamadı! Lütfen .env dosyasını kontrol edin.")
		return
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		config.Log.Error("Veritabanı bağlantısı başarısız oldu", zap.Error(err))
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		config.Log.Error("sql.DB alınamadı. ", zap.Error(err))
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

	config.Log.Debug("Veritabanına başarıyla bağlanıldı.")
	config.Log.Info("Connection Pooling ayarları uygulandı.")
}
