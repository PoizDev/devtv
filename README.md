<p align="center">
  <img src="https://i.hizliresim.com/7ndpq83.png" alt="GDG Bursa Logo" height="64">
</p>

<h1 align="center">DevTV</h1>

<p align="center">
  <code>devtv.devfestbursa.com</code> için geliştirilmiş bir etkinlik akışı sistemidir. Bu sistem production'a hazırlık açısından pek çok önlem ve özellikle bezenmiştir.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Gin-1.12-00ADD8?style=for-the-badge&logo=go" alt="Gin">
  <img src="https://img.shields.io/badge/PostgreSQL-15+-316192?style=for-the-badge&logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7 Alpine-DC382D?style=for-the-badge&logo=redis" alt="Redis">
  <img src="https://img.shields.io/badge/Docker-Distroless-2496ED?style=for-the-badge&logo=docker" alt="Docker">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
</p>

---

# GDG Bursa - DevTV

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)
![Gin Framework](https://img.shields.io/badge/Gin-v1.11.0-00ADD8?style=for-the-badge&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17+-316192?style=for-the-badge&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-alpine-DC382D?style=for-the-badge&logo=redis)
![Protobuf](https://img.shields.io/badge/Protobuf-Health-4285F4?style=for-the-badge&logo=google)
![Docker](https://img.shields.io/badge/Docker-Distroless-2496ED?style=for-the-badge&logo=docker)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)

## Kullanilan Teknolojiler

| Teknoloji | Versiyon | Kullanim |
|-----------|----------|----------|
| Go | 1.24.5 | Ana dil |
| Gin | 1.11.0 | HTTP framework |
| PostgreSQL | 17+ | Veritabani |
| Redis | alpine | Cache, fallback, WS broadcast |
| GORM | 1.31.1 | ORM |
| Protobuf | 1.36.11 | Health endpoint serializasyonu |
| Docker | distroless | Container runtime |

## Standart Sistem Kurulumu

1. Repoyu Klonlayın

```
    git clone https://github.com/poizdev/devtv.git
```

-eğer bu aşamada bi sorunla karşılaşırsanız github üzerinden klasöre indir yapabilirsiniz-

1. Bağımlılıkları Yükleyin

```
    go mod download
    go mod tidy
```

1. in/devtv.env dosyasını oluşturun (örnek ektedir.)

```
    dsn="user=kullaniciadi password=sifreniz dbname=dbadi port=5432 sslmode=disable TimeZone=Europe/Istanbul"
    JWT_SECRET="gizli keyiniz"
```

1. Uygulamayı çalıştırın

```
    go run main.go
```

## Ortam Değişkenleri (devtv.env)

Sistem, hassas veriler ve konfigürasyonu `.env` dosyasından okur. Bu dosya **asla** git'e commit edilmemelidir.

```dotenv
# PostgreSQL Bağlantı Stringi (DSN - Data Source Name)
dsn="user=postgres password=sifreniz dbname=devtv port=5432 sslmode=disable TimeZone=Europe/Istanbul"

# JWT Token İmzalama Anahtarı
JWT_SECRET="güvenli-anahtar-buraya-yazılır"
```

**DSN Parametreleri:**

- **user**: PostgreSQL kullanıcı adı (varsayılan: postgres)
- **password**: PostgreSQL şifresi
- **dbname**: Veritabanı adı (devtv)
- **port**: PostgreSQL dinleme portu (varsayılan: 5432)
- **sslmode**: SSL bağlantı modu (disable = SSL yok, production'da require olmalı)
- **TimeZone**: Zaman dilimi (Europe/Istanbul = +3 UTC)

## Docker ile Kurulum (Önerilen)

Projeyi ayağa kaldırmanın en kolay ve production ortamına en uygun yolu Docker kullanmaktır. Projede distroless (güvenli ve hafif) imaj tabanlı hazır bir `Dockerfile` ve `compose.yaml` dosyası bulunmaktadır.

### 1. Gerekli Konfigürasyon Dosyalarını Hazırlayın

Proje ana dizininde bir `.env` dosyası oluşturun (Docker Compose ortam değişkenleri için):

```dotenv
POSTGRES_USER=devtv
POSTGRES_PASSWORD=cok_guclu_db_sifreniz
POSTGRES_DB=devtv
API_PORT=2012
```

`in/devtv.env` dosyasını oluşturun (JWT sırrı vb. için):

```dotenv
JWT_SECRET="gizli-jwt-keyiniz"
```

*(Not: `compose.yaml` kullanıldığında veritabanı DSN bilgisi otomatik olarak ortam değişkenlerinden oluşturulur, `in/devtv.env` dosyasında tekrar belirtmenize gerek yoktur.)*

### 2. Uygulamayı Başlatın

Uygulamayı ve veritabanını arka planda ayağa kaldırmak için:

```bash
docker compose up -d
```

Sadece veritabanını ayağa kaldırmak isterseniz:

```bash
docker compose up -d db
```

Logları anlık takip etmek için:

```bash
docker compose logs -f api
```

### 3. Sistemi Durdurmak İçin

```bash
docker compose down
```

### Docker Yapısının Özellikleri ve Avantajları

- **Güvenlik (Distroless Image):** API konteyneri `gcr.io/distroless/static-debian12:nonroot` imajı kullanır. İçerisinde shell (`sh`, `bash`) veya gereksiz araçlar barındırmaz, böylece potansiyel saldırı yüzeyi minimuma indirilir ve izole edilmiş `nonroot` kullanıcısı ile çalışır.
- **Otomatik Sağlık Kontrolü (Healthcheck):** API konteyneri veritabanının `healthcheck` sürecinin tamamlanmasını bekler (`depends_on: condition: service_healthy`), böylece veritabanı tam hazır olmadan API başlatılmaz.
- **Kalıcı Veri (Volumes):** Veritabanı verileri ve uygulama logları Docker Volume'leri (`pgdata` ve `devtv-logs`) kullanılarak konteyner silinse bile kalıcı hale getirilir.
- **Optimize Derleme (Multi-Stage Build):** Go uygulaması derlenirken gereksiz semboller temizlenir (`-ldflags="-s -w" -trimpath`) ve uygulamanız son derece küçük ve optimize edilmiş bir konteyner haline gelir.

## API Dokümantasyonu

### Base URL

```
http://localhost:2012
```

### Kimlik Dogrulama ve Admin girisi

DevTV sistemi, arka planda JWT Token ve Role Auth sistemlerini kullanir. Sistem guvenlik nedeniyle signup uzerinden admin rolu atanmasina izin vermez. Ilk admin kullanicisi veritabanina dogrudan eklenmeli veya mevcut bir admin tarafindan UpdateUser endpoint'i ile atanmalidir.

Signup'ta izin verilen roller: `user`, `moderator`

Signup'ta izin verilmeyen roller: `admin` (403 Forbidden doner)

Admin tokeni aldiktan sonra /admin ile korunan endpointlere istek gonderebilirsiniz. Bunun icin ornek istek:

```
POST - localhost:2012/signup

{
    "username":"moderator1",
    "password":"guclusifre123",
    "role":"moderator"
}
```

Daha sonra token'ı almak için

```
POST - localhost:2012/login

{
    "username":"admin",
    "password":"çokgüçlüşifre"
}
```

Postman'inizin header kısmında Cookie olarak şunu göreceksiniz:

```
Auth=<jwt-token>; Path=/; Domain=localhost; Max-Age=2592000; HttpOnly; SameSite=None
```

Frontend çıkımında admin panel ve health controller sayfasında arka planda JWT Token'ın **Cookie olarak tutulması gerekmektedir.**

### Kullanici Yonetimi

1. Kullanici Olusturma (Signup)

```
POST - localhost:2012/signup

JSON Body Ornegi:
{
    "username": "yenikullanici",
    "password": "guclusifre123",
    "role": "user"
}

Yanit:
{
    "message": "User created successfully"
}
```

**Validasyon Kurallari:**

- `username` ve `password` zorunludur
- Sifre en az 6 karakter olmalidir
- Rol belirtilmezse otomatik olarak `user` atanir

**Roller (Role):**

- **user**: Standart kullanici, sadece okuma islemleri (signup ile alinabilir)
- **moderator**: Orta yetkili kullanici (signup ile alinabilir)
- **admin**: Tum sisteme erisim (signup ile alinamaz, sadece mevcut admin atayabilir)

1. Kullanıcı Girişi (Login)

```
POST - localhost:2012/login

JSON Body Örneği:
{
    "username": "admin",
    "password": "çokgüçlüşifre"
}

Yanıt:
{
    "message": "Login successful"
}

Cookie'ye eklenen Token:
Auth=<jwt-token>; Path=/; Domain=localhost; Max-Age=2592000; HttpOnly; SameSite=None
```

**Token Özellikleri:**

- **Süre Sonu**: 30 gün (2592000 saniye)
- **Depolama**: HttpOnly Cookie (JavaScript'ten erişilemez, güvenli)
- **Kullanım**: Tüm /admin endpoint'lerine erişim için gerekli

1. Tüm Kullanıcıları Görüntüleme (Admin)

**!!Önemli: Bu istek JWT Token gerektirmektedir.**

```
GET - localhost:2012/admin/users

Yanıt:
[
    {   
        "user_id": 1,
        "username": "admin",
        "role": "admin",
        "created_at": "2025-12-08T20:00:00+03:00"
    }
]

Not: Şifre bilgisi geri döndürülmez, güvenlik nedeniyle.
```

1. Kullanıcı Silme (Admin)

```
DELETE - localhost:2012/admin/user/:id

Yanıt:
{
    "message": "Kullanıcı başarıyla silindi",
    "user_id": 2
}
```

### Facilitator Kontrolleri

Facilitator'lar yani konuşmacıların kontrolleri için endpointler, nasıl çalıştıkları ve JSON çıktıları aşağıdaki panellerde verilmiştir

1. Konuşmacıları Görüntüleme

```
GET - localhost:2012/faciliator

Çıktı:
[
    {
        "faciliator_id": 3,
        "name": "Örnek Konuşmacı",
        "title": "GDE & Flutter Developer",
        "topic": "Örnek Konu",
        "topic_details": "Örnek Konu Detayları, bu oturum içinde neler öğreneceğiniz vb.",
        "photograph": "/public/faciliators/ornek.png",
        "created_at": "2025-12-08T23:18:14.266189+03:00",
        "updated_at": "2025-12-08T23:18:14.266189+03:00"
    }
]
```

**Facilitator Alanları:**

- **faciliator_id**: Konuşmacının benzersiz kimliği
- **name**: Konuşmacı adı (max 100 karakter)
- **title**: Unvan/Pozisyon (GDE, Android Expert vb., max 100 karakter)
- **topic**: Konu başlığı (max 200 karakter)
- **topic_details**: Konu detayları ve açıklaması (metin alanı)
- **photograph**: Konuşmacı fotoğrafının dosya yolu (string formatında)

1. Facilitator Oluşturma (Admin)

**!!Önemli: Bu istek JWT Token gerektirmektedir. Önce Login'den Auth tokenleri alın, sonra Postman Header kısmında**

```
Key: Cookie
Value: Auth=<jwt-token>; Path=/; Domain=localhost; Max-Age=2592000; HttpOnly; SameSite=None
```

**ekleyin.**

```
POST - localhost:2012/admin/create/faciliator

JSON Body Örneği:
{
    "name": "Ayşe Yılmaz",
    "title": "GDE & Flutter Developer",
    "topic": "Flutter ile Mobil Uygulama Geliştirme",
    "topic_details": "Bu oturumda Flutter framework'ünü kullanarak profesyonel mobil uygulamalar nasıl geliştirilir öğreneceğiz.",
    "photograph": "/public/faciliators/ayse.png"
}

Yanıt:
{
    "message": "Facilitator oluşturuldu: Ayşe Yılmaz"
}
```

1. Konuşmacıları Konuya Göre Filtreleme

```
GET - localhost:2012/faciliator/:topic

Örnek:
GET - localhost:2012/faciliator/Flutter

Çıktı:
[
    {
        "faciliator_id": 3,
        "name": "Ayşe Yılmaz",
        "title": "GDE & Flutter Developer",
        "topic": "Flutter",
        "topic_details": "Flutter ile Mobil Uygulama Geliştirme",
        "photograph": "/public/faciliators/ayse.png",
        "created_at": "2025-12-08T23:18:14.266189+03:00",
        "updated_at": "2025-12-08T23:18:14.266189+03:00"
    }
]
```

1. Facilitator Silme (Admin)

```
DELETE - localhost:2012/admin/faciliator/:id

Yanıt:
{
    "message": "Facilitator başarıyla silindi",
    "facilitator_id": 3
}
```

1. Facilitator Güncelleme (Admin)

```
PUT - localhost:2012/admin/faciliator/:id

JSON Body Örneği (sadece güncellemek istediğiniz alanları göndermeniz yeterli):
{
    "name": "Dr. Ayşe Yılmaz",
    "photograph": "/public/faciliators/ayse_updated.png"
}

Yanıt:
{
    "message": "Facilitator başarıyla güncellendi",
    "faciliator": {
        "faciliator_id": 3,
        "name": "Dr. Ayşe Yılmaz",
        "title": "GDE & Flutter Developer",
        "topic": "Flutter",
        "topic_details": "Flutter ile Mobil Uygulama Geliştirme",
        "photograph": "/public/faciliators/ayse_updated.png",
        "created_at": "2025-12-08T23:18:14.266189+03:00",
        "updated_at": "2025-12-10T15:30:00+03:00"
    }
}
```

1. Konuşmacıları Silme (Admin)

```
(:id kısmına silmek istediğimiz ID'nin inputu verilecektir)
DELETE - localhost:2012/admin/faciliator/:id
```

1. Facilitator güncelleme

```
Not: Bütün update parametrelerinde sadece güncellemek istediğiniz parametreyi ve değeri yazmanız yeterli olur.
PUT - localhost:2012/admin/faciliator/:id

JSON Body Örneği:
{
    "name":"Emre Hızlı",
    "photograph":"/public/faciliators/emrehizli.jpeg"
}
```

### Workshoplar ve TimeSlotlar Hk

öncelikle bu sistemin en karışık olan kısmı workshoplar ve timeslotlar arasındaki ilişki. aralarında one to many gibi bir DB ilişkisi var. üst kimlik workshoplar. time slotlar ise zamanları geldiğinde altlarına bilgileri veriyor diyebiliriz. ana mevzu timeslotlar içerisinde. timeslotlar faciliatorlara bağlı. bunlar için şöyle bir diagram verebilirim.

```
┌─────────────────────────────────────┐
│         WORKSHOPS (Parent)          │
│  ┌───────────────────────────────┐  │
│  │ workshop_id (PK)              │  │
│  │ workshop_name: "Çam Atölyesi" │  │
│  │ workshop_date: 2025-11-27     │  │
│  └───────────────────────────────┘  │
└──────────────┬──────────────────────┘
               │ One-to-Many
               │
               ↓
┌──────────────────────────────────────────┐
│    WORKSHOP_TIME_SLOTS (Children)        │
│  ┌────────────────────────────────────┐  │
│  │ slot_id (PK)                       │  │
│  │ workshop_id (FK) → Workshops       │  │
│  │ faciliator_id (FK) → Faciliators   │  │
│  │ slot_start: 13:00                  │  │
│  │ slot_end: 14:00                    │  │
│  │ slot_order: 1                      │  │
│  └────────────────────────────────────┘  │
│  ┌────────────────────────────────────┐  │
│  │ slot_id (PK)                       │  │
│  │ workshop_id (FK) → Workshops       │  │
│  │ faciliator_id (FK) → Faciliators   │  │
│  │ slot_start: 14:00                  │  │
│  │ slot_end: 15:00                    │  │
│  │ slot_order: 2                      │  │
│  └────────────────────────────────────┘  │
└──────────────┬───────────────────────────┘
               │ Many-to-One
               │
               ↓
┌──────────────────────────────────────┐
│      FACILIATORS (Referenced)        │
│  ┌────────────────────────────────┐  │
│  │ faciliator_id (PK)             │  │
│  │ name: "Ayşe Yılmaz"            │  │
│  │ title: "GDE & Flutter Dev."    │  │
│  │ topic: "Flutter Development"   │  │
│  │ topic_details: "Bu oturumda..."│  │
│  │ photograph: "https://..."      │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

Veri Akışı Diagramı:

```
┌─────────────┐
│  Workshop   │ (ID: 1, Name: "Erol Kaftanoğlu Atölyesi")
└──────┬──────┘
       │
       ├─────→ TimeSlot #1 (13:00-14:00) → Faciliator #1 (Ayşe)
       │
       ├─────→ TimeSlot #2 (14:00-15:00) → Faciliator #2 (Mehmet)
       │
       └─────→ TimeSlot #3 (15:00-16:00) → Faciliator #1 (Ayşe) ← Tekrar aynı kişiye dönebiliyor.
```

Endpointlerine dönecek olursak.

1. Workshopları Görüntüleme

```
GET - localhost:2012/workshops

Örnek JSON Çıktısı:
{
    "total": 1,
    "workshops": [
        {
            "workshop_id": 5,
            "workshop_name": "Örnek Ana Sahne",
            "workshop_date": "2025-12-08T00:00:00Z",
            "created_at": "2025-12-08T23:24:11.63826+03:00",
            "updated_at": "2025-12-08T23:24:11.63826+03:00",
            "time_slots": [
                {
                    "slot_id": 11,
                    "workshop_id": 5,
                    "faciliator_id": 3,
                    "faciliator": {
                        "faciliator_id": 3,
                        "name": "Örnek Konuşmacı",
                        "title": "GDE",
                        "topic": "Örnek Konu",
                        "topic_details": "Örnek Konu Detayları",
                        "photograph": "/public/faciliators/ornek.png"
                    },
                    "slot_start": "2025-12-09T23:18:00+03:00",
                    "slot_end": "2025-12-10T00:00:00+03:00",
                    "slot_order": 1,
                    "created_at": "2025-12-08T23:24:11.638878+03:00",
                    "updated_at": "2025-12-09T23:19:00+03:00"
                }
            ]
        }
    ]
}
```

1. Seçtiğin bir workshop'un takvimini görüntüleme

```
GET - http://localhost:2012/workshops/:id/schedule

{
    "workshop_id": 2,
    "workshop_name": "Muhammet Alihan Çabuk Atölyesi",
    "workshop_date": "2025-12-04T00:00:00Z",
    "all_slots": [
        {
            "slot_id": 7,
            "slot_start": "2025-12-09T21:00:00+03:00",
            "slot_end": "2025-12-09T23:00:00+03:00",
            "slot_order": 1,
            "faciliator": {
                "faciliator_id": 3,
                "name": "Örnek Konuşmacı",
                "topic": "Örnek Konu",
                "topic_details": "Örnek Konu Detayları",
                "photograph": "/public/faciliators/ornek.png"
            }
        }
    ],
    "total_slots": 1
}
```

1. Aktif Workshopları Görüntüleme

```
GET - localhost:2012/workshops/current

Örnek JSON Çıktısı
{
    "active_workshops": [
        {
            "slot": {
                "slot_id": 10,
                "slot_start": "2025-12-09T14:00:00+03:00",
                "slot_end": "2025-12-10T23:00:00+03:00",
                "slot_order": 1,
                "faciliator": {
                    "faciliator_id": 3,
                    "name": "Örnek Konuşmacı",
                    "topic": "Örnek Konu",
                    "topic_details": "Örnek Konu Detayları",
                    "photograph": "/public/faciliators/ornek.png"
                }
            },
            "workshop_id": 4,
            "workshop_name": "Örnek Workshop"
        }
    ],
    "total": 1
}
```

1. Sonraki Workshopları Görüntüleme

```
GET - localhost:2012/workshops/upcoming
{
    "total": 1,
    "upcoming_slots": [
        {
            "slot_id": 11,
            "workshop_name": "Örnek Ana Sahne",
            "slot_start": "2025-12-10T15:00:00+03:00",
            "slot_end": "2025-12-10T23:00:00+03:00",
            "faciliator": {
                "faciliator_id": 3,
                "name": "Örnek Konuşmacı",
                "topic": "Örnek Konu",
                "topic_details": "Örnek Konu Detayları",
                "photograph": "/public/faciliators/ornek.png"
            },
            "time_until_start": "39 dakika sonra"
        }
    ]
}
```

1. Spesifik bir workshopun altındaki slotları görüntüleme

```
GET - localhost:2012/workshop/:id/slots

Örnek JSON Çıktısı:
{
    "message": "Workshop başarıyla alındı",
    "workshop": {
        "workshop_id": 4,
        "workshop_name": "Örnek Workshop",
        "workshop_date": "2025-12-08T00:00:00Z",
        "time_slots": [
            {
                "slot_id": 10,
                "slot_start": "2025-12-09T14:00:00+03:00",
                "slot_end": "2025-12-10T23:00:00+03:00",
                "slot_order": 1
            }
        ]
    }
}
```

1. Workshop Oluşturma (Admin)

**!!Önemli: Bu istek JWT Token gerektirmektedir.**

```
POST - localhost:2012/admin/create/workshop

JSON Body Örneği:
{
    "workshop_name": "Başarılı Yazılım Mimarisi",
    "workshop_date": "2025-12-15T00:00:00Z",
    "time_slots": [
        {
            "faciliator_id": 3,
            "slot_start": "2025-12-15T13:00:00Z",
            "slot_end": "2025-12-15T14:00:00Z"
        }
    ]
}

Yanıt: 
{
    "message": "Workshop oluşturuldu: Başarılı Yazılım Mimarisi (1 slot eklendi)",
    "workshop": {
        "workshop_id": 6,
        "workshop_name": "Başarılı Yazılım Mimarisi",
        "workshop_date": "2025-12-15T00:00:00Z"
    }
}
```

1. Workshop'a Slot Ekleme (Admin)

```
POST - localhost:2012/admin/:id/addslots

JSON Body Örneği:
{
    "time_slots": [
        {
            "faciliator_id": 2,
            "slot_start": "2025-12-15T14:00:00Z",
            "slot_end": "2025-12-15T15:00:00Z"
        },
        {
            "faciliator_id": 3,
            "slot_start": "2025-12-15T15:00:00Z",
            "slot_end": "2025-12-15T16:00:00Z"
        }
    ]
}

Yanıt:
{
    "message": "Slot'lar başarıyla eklendi",
    "added_slots": 2,
    "slots": [...]
}
```

1. Workshop'u Silme (Admin)

```
DELETE - localhost:2012/admin/workshop/:id

Yanıt:
{
    "message": "Workshop ve slot'ları başarıyla silindi",
    "workshop_id": 5,
    "workshop_name": "Örnek Ana Sahne",
    "deleted_slots": 2
}

Not: Bunu sildiğinde tüm slot'ları da otomatik olarak siler.
```

1. Workshop Güncelleme (Admin)

```
PUT - localhost:2012/admin/workshop/:id

JSON Body Örneği:
{
    "workshop_name": "Yeni Workshop Adı",
    "workshop_date": "2025-12-20T00:00:00Z"
}

Yanıt:
{
    "message": "Workshop başarıyla güncellendi",
    "workshop": {
        "workshop_id": 5,
        "workshop_name": "Yeni Workshop Adı",
        "workshop_date": "2025-12-20T00:00:00Z"
    }
}
```

1. Workshop'a Gecikme Ekleme (Admin)

Eğer bir workshop'un tüm slotlarını belirli bir süre ertelemek veya erkene almak istersen bu endpoint'i kullan.

```
PUT - localhost:2012/admin/workshop/:id/delay

JSON Body Örneği (5 dakika erteleme):
{
    "delay_minutes": 5
}

JSON Body Örneği (10 dakika erkene alma):
{
    "delay_minutes": -10
}

Yanıt:
{
    "message": "Workshop 5 dakika ertelendi. 2 slot güncellendi.",
    "delay_minutes": 5,
    "updated_slots": 2
}
```

1. Slot Silme (Admin)

```
DELETE - localhost:2012/admin/slot/:id

Yanıt:
{
    "message": "Slot başarıyla silindi",
    "slot_id": 10
}
```

1. Slot Güncelleme (Admin)

Tek bir slot'u güncellemek için kullanılır. Sadece güncellemek istediğiniz alanları gönderin.

```
PUT - localhost:2012/admin/slot/:id

JSON Body Örneği:
{
    "faciliator_id": 2,
    "slot_start": "2025-12-15T16:00:00Z",
    "slot_end": "2025-12-15T17:00:00Z",
    "slot_order": 3
}

Yanıt:
{
    "message": "Slot başarıyla güncellendi",
    "slot": {
        "slot_id": 10,
        "workshop_id": 5,
        "faciliator_id": 2,
        "slot_start": "2025-12-15T16:00:00Z",
        "slot_end": "2025-12-15T17:00:00Z",
        "slot_order": 3
    }
}
```

### Sponsor Kontrolleri

1. Sponsorları Görüntüleme

```
GET - localhost:2012/sponsors

Çıktı:
[
    {
        "sponsor_id": 1,
        "sponsor_name": "Google",
        "sponsor_tier": "Partner",
        "logo": "/public/sponsors/google.png",
        "advertise_video": "/public/videos/google_ad.mp4",
        "website": "https://google.com"
    }
]
```

1. Sponsor Oluşturma (Admin)

**!!Önemli: Bu istek JWT Token gerektirmektedir.**

```
POST - localhost:2012/admin/create/sponsor

JSON Body Örneği:
{
    "sponsor_name": "Microsoft",
    "sponsor_tier": "Gümüş",
    "logo": "/public/sponsors/microsoft.png",
    "advertise_video": "/public/videos/microsoft_ad.mp4",
    "website": "https://microsoft.com"
}

Yanıt:
{
    "message": "Sponsor created successfully"
}
```

1. Sponsor Silme (Admin)

```
DELETE - localhost:2012/admin/sponsor/:id

Yanıt:
{
    "message": "Sponsor başarıyla silindi",
    "sponsor_id": 1
}
```

1. Sponsor Güncelleme (Admin)

```
PUT - localhost:2012/admin/sponsor/:id

Yanıt:
{
    "message": "Sponsor başarıyla güncellendi",
    "sponsor_id": 1
}
```

### WebSocket Endpoints

WebSocket endpoints aracılığıyla real-time veriler alabilirsiniz. Bir WebSocket bağlantısı açtığınızda, server sizin için belirlenen aralıklarla güncel verileri gönderecektir.

**WebSocket Bağlantısı Nasıl Kurulur:**

JavaScript örneği:

```javascript
const ws = new WebSocket('ws://localhost:2012/ws/current');

ws.onopen = (event) => {
    console.log('Bağlı oldunuz!');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Yeni veri geldi:', data);
};

ws.onclose = () => {
    console.log('Bağlantı kapandı');
};

ws.onerror = (error) => {
    console.error('WebSocket hatası:', error);
};
```

1. Aktif Slotlar WebSocket

```
WS - ws://localhost:2012/ws/current

Gönderilen Veri Yapısı (İlk bağlantıdan ~2 saniye sonra):
{
    "active_workshops": [
        {
            "slot": {
                "slot_id": 10,
                "slot_start": "2025-12-09T14:00:00+03:00",
                "slot_end": "2025-12-10T23:00:00+03:00",
                "faciliator": {
                    "faciliator_id": 3,
                    "name": "Örnek Konuşmacı",
                    "topic": "Örnek Konu"
                }
            },
            "workshop_id": 4,
            "workshop_name": "Örnek Workshop"
        }
    ],
    "total": 1
}

Güncelleme Sıklığı: 2 saniyede bir
```

1. Yaklaşan Slotlar WebSocket

```
WS - ws://localhost:2012/ws/upcoming

Gönderilen Veri Yapısı:
{
    "upcoming_slots": [
        {
            "slot_id": 11,
            "workshop_name": "Örnek Ana Sahne",
            "slot_start": "2025-12-10T15:00:00+03:00",
            "slot_end": "2025-12-10T23:00:00+03:00",
            "faciliator": {
                "faciliator_id": 3,
                "name": "Örnek Konuşmacı",
                "topic": "Örnek Konu"
            },
            "time_until_start": "39 dakika sonra"
        }
    ],
    "total": 1
}

Güncelleme Sıklığı: 5 saniyede bir
Not: Sadece sonraki 5 etkinlik gösterilir.
```

1. Sponsorlar WebSocket

```
WS - ws://localhost:2012/ws/sponsors

Gönderilen Veri Yapısı:
{
    "sponsors": [
        {
            "sponsor_id": 1,
            "sponsor_name": "Google",
            "sponsor_logo": "/public/sponsors/google.png",
            "sponsor_link": "https://google.com"
        }
    ],
    "total": 1
}

Güncelleme Sıklığı: 10 saniyede bir
```

1. Spesifik Workshop'un Takvimi WebSocket

```
WS - ws://localhost:2012/ws/workshop/:id/schedule

Gönderilen Veri Yapısı:
{
    "workshop_id": 2,
    "workshop_name": "Muhammet Alihan Çabuk Atölyesi",
    "workshop_date": "2025-12-04T00:00:00Z",
    "all_slots": [
        {
            "slot_id": 7,
            "slot_start": "2025-12-09T21:00:00+03:00",
            "slot_end": "2025-12-09T23:00:00+03:00",
            "slot_order": 1,
            "faciliator": {
                "faciliator_id": 3,
                "name": "Örnek Konuşmacı",
                "topic": "Örnek Konu"
            }
        }
    ],
    "total_slots": 1
}

Güncelleme Sıklığı: 5 saniyede bir
```

1. Spesifik Workshop'un Aktif Slotu WebSocket

```
WS - ws://localhost:2012/ws/:id/current

Gönderilen Veri Yapısı:
{
    "slot": {
        "slot_id": 10,
        "slot_start": "2025-12-09T14:00:00+03:00",
        "slot_end": "2025-12-10T23:00:00+03:00",
        "slot_order": 1,
        "faciliator": {
            "faciliator_id": 3,
            "name": "Örnek Konuşmacı",
            "topic": "Örnek Konu"
        }
    },
    "workshop_id": 4,
    "workshop_name": "Örnek Workshop"
}

Güncelleme Sıklığı: 2 saniyede bir
Not: Eğer aktif slot yoksa null döner.
```

||

## Middlewares ve Sistem Bileşenleri

### Health Checker - Sistem Sağlığı Monitoring

Sistem sağlığı altyapı olarak Protobuf temelli *gRPC* sistemi kullanılmıştır. bundan ötürü direkt health endpointine istek atmanız binary bir veri döndürür. Örneğin;

```
GET - localhost:2012/health

Çıktı:
"\b\u0004\u0011uB\u001f?\u0018\\ \u0005)(n\u0012O\u0018@0\u00018\u0001B\u0013\n\u0001/\u0010>\u0018T!\n\u0014?P\u000bZ\u0006\bd\u0010\u0001 \u0001b\b1 secondj\u000b\b\u0006\u0010^r\f\b\"\u0010˗\u0003"
```

biraz tuhaf... di'mi...

Onun yerine atadığımız ```format`` parametresini kullanarak JSON formatında çıktı alabiliyoruz. Örneğin:

```
GET - localhost:2012/health?format=json

Çıktı:
{
  "systemUptimeSecs": "1262",
  "ramTotalMb": "11847",
  "ramUsedMb": "752",
  "ramUsagePercent": 6.3529525808560745,
  "netBytesReceivedTotal": "22944",
  "netBytesSentTotal": "21749",
  "diskUsages": [
    {
      "path": "/",
      "totalMb": "1031018",
      "usedMb": "10877",
      "usagePercent": 1.1116039672362894
    }
  ],
  "goroutineCount": 13,
  "dbStats": {
    "maxOpenConns": 100,
    "openConns": 5,
    "inUse": 3,
    "idle": 2
  },
  "appUptime": "11 minutes",
  "timestamp": "2026-05-02T22:00:48.096520535Z",
  "cacheAge": "30.007800308s",
  "apiMetrics": {
    "totalRequests": 126,
    "totalErrors": 3,
    "errorRatePercent": 2.38,
    "successRatePercent": 97.62,
    "avgResponseTimeMs": 48.5,
    "requestsByMethod": {
      "GET": 95,
      "POST": 12,
      "PUT": 4,
      "DELETE": 2,
      "OPTIONS": 10
    }
  }
}
```

**Sistem Sağlığı Parametreleri:**

- **app_uptime**: Uygulamanın çalışma süresi (İnsan tarafından okunabilir format)
- **system_uptime_seconds**: İşletim sisteminin çalışma süresi (saniye cinsinden)
- **cpu_usage_percent**: İşlemci kullanım yüzdesi
- **ram_total_mb**: Toplam RAM (MB cinsinden)
- **ram_used_mb**: Kullanılan RAM (MB cinsinden)
- **ram_usage_percent**: RAM kullanım yüzdesi
- **net_bytes_received_total**: Ağdan alınan toplam byte
- **net_bytes_sent_total**: Ağa gönderilen toplam byte
- **disk_usages**: Tüm disklerin kullanım bilgileri (path, total_mb, used_mb, usage_percent)
- **db_connection_stats**: PostgreSQL bağlantı havuzu istatistikleri
  - **max_open_conns**: Maksimum açık bağlantı sayısı
  - **open_conns**: Aktif açık bağlantı sayısı
  - **in_use**: Kullanımdaki bağlantı sayısı
  - **idle**: Boşta duran bağlantı sayısı
- **active_websockets**: Aktif WebSocket bağlantı sayısı
- **goroutine_count**: Çalışan Go rutinleri sayısı
- **api_metrics**: API istatistikleri
  - **total_requests**: Toplam istek sayısı
  - **total_errors**: Toplam hata sayısı
  - **error_rate_percent**: Hata oranı (yüzde olarak)
  - **success_rate_percent**: Başarı oranı (yüzde olarak)
  - **avg_response_time_ms**: Ortalama yanıt süresi (milisecond olarak)
  - **requests_by_method**: Metoda göre istek sayısı (GET, POST, PUT, DELETE, OPTIONS)

**Güncelleme Sıklığı:** 1 saniyede bir
**! Not:** Health endpointleri için loglar hem Gin'in içinden hem kendi log configlerim içinden **dışında tutulmuştur**. *Ayrıca API metriklerini de **etkilememektedir.**

### Circuit Breaker - Hata Toleransı

Circuit Breaker, sistem hatalarını algılayarak otomatik olarak istekleri engelleyen bir mekanizmadır. Bu, cascade hataları (domino etkisi) önler.

```
GET - localhost:2012/circuitbreaker

Çıktı (CLOSED durumu):
{
    "state": "CLOSED",
    "failures": 0,
    "threshold": 5,
    "timeout": "30s"
}

Çıktı (OPEN durumu):
{
    "state": "OPEN",
    "failures": 8,
    "threshold": 5,
    "timeout": "30s",
    "message": "Sistem hatalarını algıladığı için istek kabul etmiyor"
}

Çıktı (HALF_OPEN durumu):
{
    "state": "HALF_OPEN",
    "failures": 5,
    "threshold": 5,
    "timeout": "30s",
    "message": "Sistem iyileşiyor, sınırlı sayıda istek kabul ediliyor"
}
```

**Circuit Breaker Durumları:**

- **CLOSED**: Normal çalışma, tüm istekler kabul edilir
- **OPEN**: Hata sayısı eşiği aştığında, istekler reddedilir
- **HALF_OPEN**: OPEN durumdan çıkış deneniyor, sınırlı istekler kabul edilir

### Rate Limiting - DDoS Koruması

Sistemin her IP adresine karşı rate limiting uygulanır. Belirli sayıda isteğin üzerine çıkılırsa, o IP'den gelen istekler geçici olarak reddedilir.

```
Headers Konfigürasyonu (conf.yaml):
RateLimit:
  Limit: 100        # Saniye başına 100 istek
  Burst: 50         # Ani pik için 50 ekstra istek
```

**Rate Limit Aşıldığında:**

```
HTTP Status: 429 Too Many Requests

Yanıt:
{
    "error": "Çok fazla istek gönderdiniz. Lütfen bir dakika bekleyin."
}
```

### Timeout Middleware - İstek Zaman Aşımı

Tüm HTTP istekleri belirli bir timeout'a sahiptir. Eğer istek bu süre içinde tamamlanmazsa otomatik olarak iptal edilir.

```
Headers Konfigürasyonu (conf.yaml):
Middleware:
  RequestTimeout: 30s    # 30 saniye timeout
```

**Timeout Aşıldığında:**

```
HTTP Status: 504 Gateway Timeout

Yanıt:
{
    "error": "İstek zaman aşımına uğradı"
}
```

## Sistem Konfigürasyonu ve Mimari

### Konfigürasyon Dosyası (conf.yaml)

Sistem tüm ayarlarını `conf.yaml` dosyasından okur. Bu dosya production ve development ortamları için özelleştirilebilir.

```yaml
server: 
  port: ":2012"
  shutdown_timeout: 30s
  log_config_path: "./log4go.json"
  env_path: "./in/devtv.env"

database:
  max_idle_conns: 25
  max_open_conns: 100
  conn_max_lifetime: 3m
  conn_max_idle_time: 30s

middleware:
  circuit_breaker:
    threshold: 15
    timeout: 30s
  rate_limit:
    burst: 25
    limit: 50
  request_timeout: 5m

auth:
  cookie_domain: "localhost"       # Production: "devfestbursa.com"
  cookie_secure: false             # Production: true (HTTPS zorunlu)
  token_expiry_days: 30

cors:
  allow-origins:
    - "https://api.devfestbursa.com"
    - "https://www.api.devfestbursa.com"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "PATCH"
    - "DELETE"
    - "OPTIONS"
  allowed_headers:
    - "Origin"
    - "Content-Type"
    - "Authorization"
    - "Accept"
  expose_headers:
    - "Content-Length"
    - "Set-Cookie"
  allow_credentials: true
  max_age: 12h

redis:
  redis_url: "redis:6379"
  redis_pwr: ""
  db: 0
```

### Connection Pooling - Veritabani Baglanti Havuzu

DevTV, PostgreSQL baglantilerini verimli kullanmak icin **Connection Pooling** mekanizmasi kullanir.

```
Connection Pool Yapisi:

Pool Yonetimi:
- max_idle_conns: 25     - En fazla 25 bosta baglanti tutulur
- max_open_conns: 100    - En fazla 100 acik baglanti
- conn_max_lifetime: 3m  - Her baglanti maksimum 3 dakika acik kalabilir
- conn_max_idle_time: 30s - 30 saniyeden fazla bosta baglanti kapatilir
```

### Redis Fallback Cache - Yuksek Erisilebilirlik

HTTP GET istekleri icin iki katmanli bir cache mekanizmasi bulunur. Bu mekanizma veritabani cokse bile stale veri sunarak sistemin ayakta kalmasini saglar.

```
Iki Katmanli Cache Stratejisi:

Katman 1: devtv:cache:/endpoint   (TTL: 5 saniye)  - Taze veri
Katman 2: devtv:fallback:/endpoint (TTL: 1 saat)    - Stale veri (DB cokme durumu)

Istek Akisi:
1. Redis cache key kontrol et (2s timeout)
2. Cache HIT  -> Aninda don (~1ms)
3. Cache MISS -> Controller'a git (3s DB timeout)
4. Controller basarili -> Response + her iki key'e yaz (Pipeline)
5. Controller basarisiz / DB timeout -> Fallback key'den stale veri sun
6. Fallback'te de veri yok -> Controller hatasini ilet

Fallback Aktif Oldugunda:
- HTTP Status: 200 OK (istemci farki anlamaz)
- Header: X-Cache-Fallback: true
- Header: X-Cache-Source: redis-stale
```

Bu mekanizma `bufferedWriter` kullanarak response'u tamponlar. Controller hata donerse istemciye hicbir veri gitmeden Redis fallback devreye girer.

**Monitorlama:**

```
GET /health endpoint'inden pool istatistikleri:

{
    "db_connection_stats": {
        "max_open_conns": 50,      # Maksimum açık bağlantı
        "open_conns": 8,           # Şu anda açık bağlantı sayısı
        "in_use": 3,               # İşlemde olan bağlantılar
        "idle": 5                  # Boşta duran bağlantılar
    }
}
```

### Circuit Breaker Pattern - Kaskad Hata Önleme

Circuit Breaker, bir elektrik devresi gibi çalışan **hata toleransı desenidir**. Sistem hatalarını algılayarak otomatik olarak istekleri engeller ve servisin tamamen çökmesini önler.

```
Circuit Breaker Durumları:

1. CLOSED (Normal Çalışma) ───[Hata sayısı eşiğe ulaştı]──→ OPEN
   │
   └─ Tüm istekler işlenir
   └─ Hata sayacı tutulur
   └─ Başarılı istekler hata sayacını sıfırlar


2. OPEN (Servis Hataları Algılandı) ───[Timeout geçti]──→ HALF-OPEN
   │
   └─ Tüm istekler reddedilir
   └─ Yanıt: "HTTP 503 Service Unavailable"
   └─ Servisin iyileşmesini beklenir (timeout: 30s)


3. HALF-OPEN (Test Modu) ───[1 istek başarılı]──→ CLOSED
                         └──[1 istek başarısız]──→ OPEN
   │
   └─ Sınırlı sayıda istek denetilebilir
   └─ Servis iyileşip iyileşmediği test edilir
   └─ Başarılı sonuç → CLOSED, başarısız → OPEN
```

**Config'deki Circuit Breaker Ayarları:**

```yaml
circuit_breaker:
  threshold: 15      # 15 hatadan sonra OPEN olur
  timeout: 30s       # 30 saniye sonra HALF-OPEN'a geç
```

**Monitorlama:**

```
GET /circuitbreaker endpoint'i:

CLOSED durumu (Normal):
{
    "state": "CLOSED",
    "failures": 2,
    "threshold": 15,
    "timeout": "30s"
}

OPEN durumu (Hata):
{
    "state": "OPEN",
    "failures": 16,
    "threshold": 15,
    "timeout": "30s",
    "message": "Sistem hatalarını algıladığı için istek kabul etmiyor"
}
```

**Circuit Breaker'ın Avantajları:**

- ✅ **Cascade Failure Önleme**: Bir serviste sorun varsa, tüm sisteme yayılmaz
- ✅ **Hızlı Başarısızlık**: Sorunlu servise istek gönderilmeye çalışılmaz
- ✅ **Otomatik İyileşme**: HALF-OPEN modu ile servisteki sorun çözüldüğünde otomatik devam eder
- ✅ **Sistem Direnci**: İçsel hataların dış etkilere yansımasını engeller

### Rate Limiting - DDoS ve Overload Koruması

Rate Limiting, her IP adresinin yapabileceği istek sayısını sınırlandırır. Bu, sistem overload'ından ve DDoS saldırılarından korur.

```
Rate Limiter Mekanizması (Token Bucket Algoritması):

IP: 192.168.1.100
┌──────────────────────────────────────┐
│        Token Bucket (Kapasite: 10)   │
├──────────────────────────────────────┤
│  Burst Capacity: 5 (ani yoğunluk)    │
│  Rate: 10 istek/saniye               │
│                                      │
│  Şu anki Tokenlar: ●●●●● (5/10)      │
│                                      │
└──────────────────────────────────────┘
        │
        ├─ Her istek 1 token tüketir
        ├─ Token boşsa → İstek reddedilir (429)
        └─ Her saniye 10 token eklenir
```

**Config'deki Rate Limit Ayarları:**

```yaml
rate_limit:
  burst: 5    # Bir anda 5 ekstra istek yapılabilir (ani pik)
  limit: 10   # Ortalama 10 istek/saniye
```

**Mekanizma Detayları:**

- **Limit**: Normal hız (10 istek/saniye)
- **Burst**: Ani artışlar için ekstra kapasite (5 istek)
- **Token Bucket**: Tokenler akar gibi hızda eklenir

**Limit Aşıldığında:**

```
HTTP 429 Too Many Requests

{
    "error": "Çok fazla istek",
    "message": "Lütfen bir süre bekleyip tekrar deneyin",
    "retry_after": "1 saniye"
}
```

**Rate Limiting Faydaları:**

- ✅ **DDoS Koruması**: Saldırı trafiğini sınırlandırır
- ✅ **Adil Kaynak Dağılımı**: Bir istemcinin sistemi tekellemesini engeller
- ✅ **API Stabilizesi**: Ani trafik artışlarından korunur
- ✅ **IP Bazlı**: Her IP için ayrı limiter (spoof'lanmayı zorlaştırır)

### Request Timeout - İstek İşleme Süresi Limiti

Tüm HTTP istekleri belirli bir süre içinde tamamlanmalıdır. Aşılırsa otomatik olarak iptal edilir.

```
Request Lifecycle:

t=0s        ┌─────────────────────────────────────┐
            │ Request başlangıcı                  │
            │ Timeout counter başlar              │
            │                                     │
t=1s        │ [Processing...]                     │
            │ Database query çalışıyor            │
            │                                     │
t=2s        │ [Processing...]                     │
            │ Business logic çalışıyor            │
            │                                     │
t=5m (300s) └─────────────────────────────────────┘
            │ TIMEOUT AŞILDI!
            │ HTTP 504 Gateway Timeout
            │ Response gönderilir
```

**Config'deki Timeout Ayarı:**

```yaml
request_timeout: 5m  # 5 dakika = 300 saniye
```

**Timeout Aşıldığında:**

```
HTTP 504 Gateway Timeout

{
    "error": "İstek zaman aşımına uğradı"
}
```

### Metrics & Monitoring - Sistem Performans İstatistikleri

Sistem tüm istekleri takip eder ve performans metriklerini toplaya.

```
Toplanan Metrikler:

1. İstek Sayıları
   - TotalRequests: 15,243 (toplam)
   - RequestsByMethod: {
       "GET": 8,500,
       "POST": 4,200,
       "PUT": 1,800,
       "DELETE": 743
     }

2. Performans Verileri
   - TotalResponseTimeMs: 4,521,000ms
   - AvgResponseTime: 297ms (ortalama)
   - SlowRequests: 42 (>500ms)

3. Hata İstatistikleri
   - TotalErrors: 245
   - ErrorRate: 1.60%
   - SuccessRate: 98.40%
```

**Metrikleri Thread-Safe Toplama:**

Sistem atomic operasyonlar kullanarak metrikleri güvenli şekilde toplar (lock olmadan):

```go
// Atomic - Lock yok, çok hızlı ✅
atomic.AddInt64(&TotalRequests, 1)
atomic.LoadInt64(&TotalRequests)

// vs. Mutex - Lock var, yavaş ❌
mutex.Lock()
requests++
mutex.Unlock()
```

**Metriklerin Faydaları:**

- ✅ **Performans Monitoring**: Ortalama response time takip
- ✅ **Hata Tespiti**: Hata oranı yüksekse alert
- ✅ **Yavaş Request Tespit**: 500ms+ istekler loglanır
- ✅ **Kapasite Planlama**: Method'lara göre trafik analizi

## Request Lifecycle - İstek İçinden Geçtiği Aşamalar

Bir HTTP isteği sistemde şu aşamaları takip eder:

```
┌──────────────┐
│  Client      │ (Tarayıcı, Postman, Mobile App)
└──────┬───────┘
       │
       │ HTTP Request
       ↓
┌──────────────────────────────────────────┐
│  CORS Middleware                         │
│  ✓ Origin kontrolü                       │
│  ✓ Method kontrolü                       │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Rate Limiter Middleware                 │
│  ✓ IP bazlı limit kontrolü               │
│  ✗ Aşıldıysa 429 + Abort                │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Circuit Breaker Middleware              │
│  ✓ Sistem durumu kontrolü                │
│  ✗ OPEN ise 503 + Abort                 │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Timeout Middleware                      │
│  ✓ Context deadline set (5 dakika)      │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Auth Middleware (if /admin/*/)         │
│  ✓ JWT Token kontrolü                   │
│  ✓ Admin role kontrolü                  │
│  ✗ Yoksa 401/403 + Abort                │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Metrics Middleware                      │
│  ✓ İstek sayısı artır (atomic)         │
│  ✓ İstek zamanı kaydet                 │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Request Logger Middleware               │
│  ✓ Tüm istek detaylarını logla          │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Handler (Controller)                    │
│  ✓ Business Logic Çalışır                │
│  • Database sorgusu                      │
│  • Veri işlemesi                        │
│  • Response hazırlanması                │
└──────┬───────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────────┐
│  Response                                │
│  ✓ JSON + HTTP Status Code              │
│  ✓ Circuit Breaker'a durumu bildir     │
│  ✓ Metrikleri güncelle                 │
└──────┬───────────────────────────────────┘
       │
       │ HTTP Response
       ↓
┌──────────────┐
│  Client      │
└──────────────┘
```

### Graceful Shutdown - Güvenli Kapatma

Sistem kapatılırken tüm açık istekleri tamamlamak için **Graceful Shutdown** mekanizması kullanılır.

```
Normal Shutdown Süreci:

t=0s    ┌─────────────────────────────────┐
        │ SIGTERM / SIGINT alındı         │
        │ (Ctrl+C veya sistem sinyali)   │
        │                                 │
        │ Server yeni istekleri kabul     │
        │ etmeyi durdurur                │
        └──────────────┬──────────────────┘
                       │
t=0.1s  ┌──────────────▼──────────────────┐
        │ Açık istekler işlemeye devam    │
        │ • Query'ler tamamlanır         │
        │ • Response'lar gönderilir      │
        │ • WebSocket bağlantıları kapat │
        └──────────────┬──────────────────┘
                       │
t=30s   ┌──────────────▼──────────────────┐
        │ Shutdown timeout (30s) tamamlandı
        │ Kalan istekler iptal edilir    │
        │                                 │
        │ Veritabanı bağlantıları kapat  │
        │ Log dosyaları flush edilir     │
        │ Sistem kapatılır               │
        └─────────────────────────────────┘
```

**Config'deki Shutdown Ayarı:**

```yaml
server:
  shutdown_timeout: 30s  
```

**Faydaları:**

- ✅ **Veri Kaybı Önleme**: Açık transactionlar tamamlanır
- ✅ **Bağlantı Kapatma**: Tüm bağlantılar düzgün kapatılır
- ✅ **Temiz Çıkış**: Log ve cache'ler düzgün kapatılır

## Dosya Yükleme (Frontend Asset'leri)

Fotoğraflar ve diğer dosyalar aşağıdaki klasörlere yerleştirilmelidir:

```
public/
├── faciliators/        # Konuşmacı fotoğrafları
│   ├── ornek.png
│   └── emrehizli.jpeg
├── sponsors/           # Sponsor logoları
│   ├── google.png
│   └── microsoft.png
└── videos/             # Video dosyaları
    └── workshop1.mp4
```

Veritabanında dosya yolunu şu şekilde belirtmelisiniz:

```
/public/faciliators/ornek.png
/public/sponsors/google.png
```

## Hata Kodları ve Anlamları

| HTTP Kodu | Anlamı | Açıklama |
|-----------|--------|----------|
| 200 | OK | İstek başarıyla tamamlandı |
| 201 | Created | Yeni kayıt oluşturuldu |
| 400 | Bad Request | Gönderilen veriler yanlış veya eksik |
| 401 | Unauthorized | JWT Token bulunamadı veya geçersiz |
| 403 | Forbidden | Yetkisiz erişim (Admin yetkisi gerekli) |
| 404 | Not Found | İstenen kayıt bulunamadı |
| 429 | Too Many Requests | Rate limit aşıldı, istek reddedildi |
| 503 | Service Unavailable | Circuit Breaker AÇIK, servis kullanılamıyor |
| 504 | Gateway Timeout | İstek timeout'a uğradı |

## İletişim ve Destek

Sistemle ilgili sorularınız veya bulduğunuz hatalar için lütfen GitHub Issues'de bir issue açınız.

GitHub: <https://github.com/poizdev/devtv>

Mail: <musa@gdgbursa.com>

**Son Guncelleme:** Mayis 09, 2026
