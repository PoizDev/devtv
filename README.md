# GDG Bursa - DevTV

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![Gin Framework](https://img.shields.io/badge/Gin-v1.9.1-00ADD8?style=for-the-badge&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-316192?style=for-the-badge&logo=postgresql)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)

**DevFest Bursa 2025 için gerçek zamanlı workshop programlama ve yönetim sistemi**

[Özellikler](#-özellikler) • [Hızlı Başlangıç](#-hızlı-başlangıç) • [API Dokümantasyonu](#-api-dokümantasyonu) • [Mimari](#-mimari)

</div>

---

## İçindekiler

- [Genel Bakış](#-genel-bakış)
- [Özellikler](#-özellikler)
- [Teknoloji Yığını](#-teknoloji-yığını)
- [Hızlı Başlangıç](#-hızlı-başlangıç)
- [Proje Yapısı](#-proje-yapısı)
- [API Dokümantasyonu](#-api-dokümantasyonu)
- [Veritabanı Şeması](#-veritabanı-şeması)
- [Middleware & Pattern'ler](#-middleware--patternler)
- [Yapılandırma](#-yapılandırma)
- [Geliştirme](#-geliştirme)
- [Deployment](#-deployment)
- [Katkıda Bulunma](#-katkıda-bulunma)
- [Lisans](#-lisans)

---

## Genel Bakış

DevFest Bursa Workshop Yönetim API'si, DevFest etkinlikleri sırasında birden fazla eş zamanlı workshop yönetimi, gerçek zamanlı program takibi ve kusursuz katılımcı deneyimi için geliştirilmiş production-ready bir RESTful API'dir.

### Temel Vurgular

- **Gerçek zamanlı takip** - Mevcut ve yaklaşan workshop slotlarını anlık izleme
- **Dinamik programlama** - Canlı gecikme yönetimi ile esnek zamanlama
- **Çoklu workshop desteği** - Paralel oturum yönetimi
- **Güvenli admin paneli** - JWT kimlik doğrulama ile
- **Production-ready** - Graceful shutdown, circuit breaker ve timeout'lar
- **Yüksek performans** - Connection pooling ve optimize edilmiş sorgular

---

## Özellikler

### Genel Kullanıcı Özellikleri
- Tüm workshop'ları ve programları görüntüleme
- Konuşmacıları ve konularını inceleme
- Etkinlik sponsorlarını görüntüleme
- **Gerçek zamanlı mevcut slot takibi** - Tüm workshop'larda şu anda ne oluyor?
-  **Yaklaşan oturumlar önizlemesi** - Geri sayım sayaçları ile
- Mobil uyumlu JSON yanıtları

### Admin Özellikleri
- Kullanıcı yönetimi ve kimlik doğrulama
- Birden fazla zaman dilimi ile workshop oluşturma
- Mevcut workshop'lara slot ekleme
- **Workshop programlarını dinamik olarak erteleme/öne alma**
- Workshop canlı durumunu değiştirme
- Konuşmacı ve sponsor yönetimi
- **Cascade delete** - Workshop'ları ilişkili verilerle temizleme

---

## Teknoloji Yığını

### Temel Teknolojiler
- **Dil:** Go 1.21+
- **Framework:** Gin Web Framework
- **Veritabanı:** PostgreSQL 15+
- **ORM:** GORM v2

### Kütüphaneler & Araçlar
- **Kimlik Doğrulama:** JWT (golang-jwt/jwt)
- **CORS:** gin-contrib/cors
- **Loglama:** log4go
- **Environment:** godotenv
- **Şifre Hashleme:** bcrypt

### Pattern'ler & Mimari
- RESTful API tasarımı
- DTO (Data Transfer Objects)
- Middleware pattern
- Transaction yönetimi
- Circuit Breaker pattern
- Graceful shutdown
- Connection pooling

---

## Hızlı Başlangıç

### Gereksinimler

- Go 1.21 veya üzeri
- PostgreSQL 15+
- Git

### Kurulum

1. **Repoyu klonlayın**
   ```bash
   git clone https://github.com/poizdev/devfest-bursa-api.git
   cd devfest-bursa-api
   ```

2. **Bağımlılıkları yükleyin**
   ```bash
   go mod download
   ```

3. **Environment değişkenlerini ayarlayın**
   
   `in/devtv.env` dosyasını oluşturun:
   ```env
   dsn=host=localhost user=postgres password=sifreniz dbname=devfest port=5432 sslmode=disable TimeZone=Europe/Istanbul
   ```

4. **Loglama konfigürasyonunu ayarlayın**
   
   `log4go.json` dosyasını oluşturun:
   ```json
   {
     "console": {
       "level": "INFO",
       "pattern": "[%D %T] [%L] %M"
     }
   }
   ```

5. **Uygulamayı çalıştırın**
   ```bash
   go run main.go
   ```

   Sunucu `http://localhost:2012` adresinde başlayacak

6. **API'yi test edin**
   ```bash
   curl http://localhost:2012/ping
   # Yanıt: {"message":"pong"}
   ```

---

## Proje Yapısı

```
devtv/
├── main.go                 # Uygulama giriş noktası
├── controllers/            # İstek işleyicileri
│   ├── authcontroller.go   # Kimlik doğrulama
│   ├── usercontroller.go   # Kullanıcı CRUD
│   ├── workshopcontroller.go # Workshop yönetimi
│   ├── faciliatorcontroller.go
│   └── sponsorcontroller.go
├── middlewares/            # HTTP middleware'leri
│   ├── middlewares.go      # Auth & Timeout
│   └── circuitbreaker.go   # Circuit breaker pattern
├── models/                 # Veritabanı modelleri & DTO'lar
│   ├── user.go
│   ├── workshops.go
│   ├── faciliators.go
│   └── sponsors.go
├── in/                     # Altyapı
│   ├── connect.go          # Veritabanı bağlantısı
│   └── syncdb.go           # Auto-migration
├── log4go.json            # Loglama konfigürasyonu
└── go.mod                 # Bağımlılıklar
```

---

## API Dokümantasyonu

### Base URL
```
http://localhost:2012
```

### Kimlik Doğrulama
Admin endpoint'leri cookie'de JWT token gerektirir:
```
Cookie: Auth=<jwt_token>
```

---

### Genel Endpoint'ler

#### Sağlık Kontrolü & Durum

```http
GET /ping
GET /circuitbreaker
```

**Örnek Yanıt:**
```json
{
  "status": "ok",
  "circuit_breaker": "CLOSED",
  "failures": 0
}
```

---

#### Kimlik Doğrulama

```http
POST /signup
POST /login
```

**Kayıt İsteği:**
```json
{
  "username": "ahmetyilmaz",
  "email": "ahmet@example.com",
  "password": "güvenlişifre123"
}
```

**Giriş Yanıtı:**
```json
{
  "message": "Giriş başarılı",
  "user": {
    "user_id": 1,
    "username": "ahmetyilmaz",
    "email": "ahmet@example.com",
    "role": "user"
  }
}
```

---

#### Workshop'lar

```http
GET /workshops                  # Tüm workshop'ları listele
GET /workshops/:id/schedule     # Belirli bir workshop'un programı
GET /workshops/current          # Şu anda aktif olan slot'lar
GET /workshops/upcoming         # Yaklaşan slot'lar
```

**Mevcut Workshop'lar Yanıtı:**
```json
{
  "current_slots": [
    {
      "workshop_id": 1,
      "workshop_name": "Çam Atölyesi",
      "slot_id": 5,
      "slot_start": "2025-11-28T09:30:00Z",
      "slot_end": "2025-11-28T10:00:00Z",
      "faciliator": {
        "name": "Ahmet Yılmaz",
        "topic": "Flutter ile Android Development",
        "photograph": "/public/faciliators/ahmet.png"
      }
    }
  ],
  "total": 1
}
```

**Yaklaşan Slot'lar Yanıtı:**
```json
{
  "upcoming_slots": [
    {
      "slot_id": 6,
      "workshop_name": "Fidan Atölyesi",
      "slot_start": "2025-11-28T10:00:00Z",
      "slot_end": "2025-11-28T10:30:00Z",
      "time_until_start": "15 dakika sonra",
      "faciliator": {...}
    }
  ],
  "total": 5
}
```

---

#### Konuşmacılar & Sponsorlar

```http
GET /faciliator    # Tüm konuşmacıları listele
GET /sponsors      # Tüm sponsorları listele
```

---

### Admin Endpoint'leri

Tüm admin endpoint'leri kimlik doğrulama ve admin rolü gerektirir.

#### Kullanıcı Yönetimi

```http
GET /admin/users   # Tüm kullanıcıları listele
```

---

#### Workshop Yönetimi

**Slot'larla Workshop Oluştur:**
```http
POST /admin/workshops/create
```

**İstek Gövdesi:**
```json
{
  "workshop_name": "Çam Atölyesi",
  "workshop_date": "2025-11-28T00:00:00Z",
  "time_slots": [
    {
      "faciliator_id": 1,
      "slot_start": "2025-11-28T09:00:00Z",
      "slot_end": "2025-11-28T09:30:00Z"
    },
    {
      "faciliator_id": 2,
      "slot_start": "2025-11-28T09:30:00Z",
      "slot_end": "2025-11-28T10:00:00Z"
    }
  ]
}
```

**Mevcut Workshop'a Slot Ekle:**
```http
POST /admin/workshops/:id/slots
```

**Workshop Programını Ertele/Öne Al:**
```http
PUT /admin/workshops/:id/delay
```

**İstek Gövdesi:**
```json
{
  "delay_minutes": 15  // Pozitif = erteleme, Negatif = öne alma
}
```

**Örnek Kullanım:**
- `"delay_minutes": 10` → 10 dakika ertele
- `"delay_minutes": -5` → 5 dakika öne al

**Workshop Canlı Durumunu Değiştir:**
```http
PUT /admin/workshops/:id/live
```

**İstek Gövdesi:**
```json
{
  "is_live": true
}
```

**Workshop'u Sil:**
```http
DELETE /admin/workshops/:id
```

Tüm ilişkili zaman dilimlerini otomatik olarak siler (cascade delete).

---

#### Konuşmacı Yönetimi

```http
POST /admin/create/faciliator
```

**İstek Gövdesi:**
```json
{
  "name": "Ahmet Yılmaz",
  "topic": "Flutter Development",
  "topic_details": "Flutter ile cross-platform mobil uygulama geliştirme",
  "photograph": "/public/faciliators/ahmet.png"
}
```

---

#### Sponsor Yönetimi

```http
POST /admin/create/sponsor
```

**İstek Gövdesi:**
```json
{
  "name": "Google",
  "logo": "/public/sponsors/google.png",
  "website": "https://google.com",
  "tier": "platinum"
}
```

---

## Fotoğraf Yönetimi

### Statik Dosya Servisi

API, fotoğrafları `/public` dizininden servis eder.

**Dizin Yapısı:**
```
public/
├── faciliators/
│   ├── ahmet.png
│   ├── mehmet.jpg
│   └── ayse.png
└── sponsors/
    ├── google.png
    ├── microsoft.png
    └── aws.png
```

### Fotoğraf Ekleme

#### 1️⃣ **Manuel Yükleme (Basit Yöntem)**

```bash
# Konuşmacı fotoğrafı ekle
cp ahmet.png public/faciliators/

# Sponsor logosu ekle
cp google.png public/sponsors/
```

#### 2️⃣ **API ile Yükleme (Gelişmiş)**

**Endpoint Ekle (main.go):**
```go
// Statik dosya servisi
r.Static("/public", "./public")

// Fotoğraf yükleme endpoint'i
admin.POST("/upload/faciliator", controllers.UploadFaciliatorPhoto)
admin.POST("/upload/sponsor", controllers.UploadSponsorLogo)
```

**Controller Örneği:**
```go
func UploadFaciliatorPhoto(c *gin.Context) {
    file, err := c.FormFile("photo")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Dosya yüklenemedi"})
        return
    }

    // Dosya adını oluştur
    filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
    filepath := fmt.Sprintf("./public/faciliators/%s", filename)

    // Dosyayı kaydet
    if err := c.SaveUploadedFile(file, filepath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Dosya kaydedilemedi"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Fotoğraf yüklendi",
        "url":     "/public/faciliators/" + filename,
    })
}
```

**Postman ile Test:**
```
POST /admin/upload/faciliator
Headers:
  Cookie: Auth=<token>
Body:
  form-data
  Key: photo
  Value: [Dosya Seç]
```

### Fotoğraf Kullanımı

**Konuşmacı Oluştururken:**
```json
{
  "name": "Ahmet Yılmaz",
  "topic": "Flutter",
  "photograph": "/public/faciliators/ahmet.png"
}
```

**Frontend'de Gösterme:**
```html
<img src="http://localhost:2012/public/faciliators/ahmet.png" alt="Ahmet Yılmaz">
```

### Fotoğraf Boyutlandırma (Opsiyonel)

**Image Processing Kütüphanesi Ekle:**
```bash
go get github.com/disintegration/imaging
```

**Otomatik Resize:**
```go
import "github.com/disintegration/imaging"

func resizeImage(src, dst string) error {
    img, err := imaging.Open(src)
    if err != nil {
        return err
    }

    // 400x400 boyutuna küçült
    resized := imaging.Resize(img, 400, 400, imaging.Lanczos)
    
    return imaging.Save(resized, dst)
}
```

---

## Veritabanı Şeması

### Users Tablosu
```sql
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Workshops Tablosu
```sql
CREATE TABLE workshops (
    workshop_id SERIAL PRIMARY KEY,
    workshop_name VARCHAR(100) NOT NULL,
    workshop_date DATE NOT NULL,
    is_live BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

### Workshop Time Slots Tablosu
```sql
CREATE TABLE workshop_time_slots (
    slot_id SERIAL PRIMARY KEY,
    workshop_id INTEGER REFERENCES workshops(workshop_id),
    faciliator_id INTEGER REFERENCES faciliators(faciliator_id),
    slot_start TIMESTAMP NOT NULL,
    slot_end TIMESTAMP NOT NULL,
    slot_order INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

### Facilitators Tablosu
```sql
CREATE TABLE faciliators (
    faciliator_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    topic VARCHAR(200),
    topic_details TEXT,
    photograph VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Sponsors Tablosu
```sql
CREATE TABLE sponsors (
    sponsor_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    logo VARCHAR(255),
    website VARCHAR(255),
    tier VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Middleware & Pattern'ler

### Authentication Middleware
- JWT token doğrulama
- Admin rolü kontrolü
- Cookie tabanlı kimlik doğrulama

### Timeout Middleware
- İstek zaman aşımı: 5 dakika
- Uzun süren istekleri önler
- 504 Gateway Timeout döner

### Circuit Breaker Pattern
- **Eşik:** 5 hata
- **Timeout:** 30 saniye
- **Durumlar:** CLOSED → OPEN → HALF-OPEN
- Zincirleme hatalara karşı koruma

### Graceful Shutdown
- 30 saniyelik kapatma zaman aşımı
- Aktif isteklerin tamamlanmasını bekler
- Veritabanı bağlantılarını temiz kapatır

### Connection Pooling
```go
MaxIdleConns:     15    // Boşta tutulacak bağlantı sayısı
MaxOpenConns:     50    // Maksimum açık bağlantı
ConnMaxLifetime:  5 dakika
ConnMaxIdleTime:  1 dakika
```

---

## Yapılandırma

### Environment Değişkenleri

| Değişken | Açıklama | Örnek |
|----------|----------|-------|
| `dsn` | PostgreSQL bağlantı string'i | `host=localhost user=postgres...` |

### CORS Yapılandırması (Production)

```go
AllowOrigins:     []string{
    "https://devfestbursa.com",
    "https://www.devfestbursa.com"
}
AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "Cookie"}
ExposeHeaders:    []string{"Content-Length", "Set-Cookie"}
AllowCredentials: true
MaxAge:           12 * time.Hour
```

---

## Geliştirme

### Development Modunda Çalıştır
```bash
go run main.go
```

### Testleri Çalıştır
```bash
go test ./...
```

### Production için Build Et
```bash
go build -o devfest-api main.go
```

### Kodu Formatla
```bash
go fmt ./...
```

### Kodu Lint'le
```bash
golangci-lint run
```

---

## Deployment

### Docker Deployment

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o devfest-api main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/devfest-api .
COPY --from=builder /app/log4go.json .
COPY --from=builder /app/public ./public
EXPOSE 2012
CMD ["./devfest-api"]
```

**Build & Run:**
```bash
docker build -t devfest-api .
docker run -p 2012:2012 --env-file .env devfest-api
```

---

### Systemd Servisi

**`/etc/systemd/system/devtv.service`:**
```ini
[Unit]
Description=DevTV
After=network.target postgresql.service

[Service]
Type=simple
User=devfest
WorkingDirectory=/opt/devfest-api
ExecStart=/opt/devfest-api/devfest-api
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

**Etkinleştir & Başlat:**
```bash
sudo systemctl enable devfest-api
sudo systemctl start devfest-api
sudo systemctl status devfest-api
```

---

## Katkıda Bulunma

Katkılarınızı bekliyoruz! Lütfen şu adımları izleyin:

1. Repo'yu fork'layın
2. Feature branch'i oluşturun (`git checkout -b feature/harika-ozellik`)
3. Değişikliklerinizi commit'leyin (`git commit -m 'Harika özellik eklendi'`)
4. Branch'inizi push'layın (`git push origin feature/harika-ozellik`)
5. Pull Request açın

### Commit Kuralları
```
feat: Yeni özellik eklendi
fix: Bug düzeltildi
docs: Dokümantasyon güncellendi
style: Kod formatlandı
refactor: Kod yeniden yapılandırıldı
test: Test eklendi
chore: Bağımlılıklar güncellendi
```

---

## Lisans

Bu proje MIT Lisansı altında lisanslanmıştır - detaylar için [LICENSE](LICENSE) dosyasına bakın.

---

## Ekip

**Geliştirici:** Musa Efe KOBAK ([@poizdev](https://github.com/poizdev))

**Etkinlik:** DevFest Bursa 2025

**Organizasyon:** GDG Bursa

---

## Teşekkürler

- Gin Web Framework ekibine
- GORM katkıda bulunanlara
- DevFest Bursa tasarım ekibine
- Tüm açık kaynak katkıda bulunanlara

---

## Destek

- **Issue'lar:** [GitHub Issues](https://github.com/poizdev/devtv/issues)
- **E-posta:** musaefekobak@gmail.com
- **Website:** [devfestbursa.com](https://devfestbursa.com) & [gdgbursa.com](https://gdgbursa.com)

---

<div align="center">

**DevFest Bursa 2025 için ❤️ ile yapıldı**

[![Bu repo'yu yıldızla](https://img.shields.io/github/stars/poizdev/devtv?style=social)](https://github.com/poizdev/devfest-bursa-api)

</div>