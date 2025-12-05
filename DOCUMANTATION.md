# DevTV Sistem Dokümantasyonu

## Genel Bakış
DevTV, canlı etkinlik veya konferans formatı için atölye çalışmalarını, konuşmacıları (facilitators), sponsorları ve kullanıcıları yönetmek üzere tasarlanmış bir arka uç (backend) sistemidir. Yönetim için bir REST API ve programların ve aktif oturumların gerçek zamanlı güncellemeleri için WebSocket uç noktaları sağlar.

Sistem, **Gin** web çerçevesi ile **Go (Golang)** kullanılarak oluşturulmuştur. Veritabanı etkileşimleri için **GORM** kullanır.

## Mimari Yapı

Proje, standart bir MVC benzeri (Model-View-Controller) yapıyı takip eder; ayrıca middleware (ara katman) ve websocket işlemleri için katmanlar içerir.

-   **`models/`**: Veritabanı şemasını ve veri yapılarını tanımlar.
-   **`controllers/`**: HTTP isteklerini ve WebSocket bağlantılarını işleyen mantığı içerir.
-   **`middlewares/`**: Loglama, yetkilendirme ve hız sınırlama (rate limiting) gibi kesişen endişeler için özel ara katman yazılımlarını içerir.

---

## Ara Katman Yazılımları (Middlewares)

Sistem, kararlılık, güvenlik ve gözlemlenebilirlik sağlamak için çeşitli middleware'ler kullanır.

### 1. Devre Kesici (Circuit Breaker) - `circuitbreaker.go`
-   **Amaç**: Bir servis (veya veritabanı) ağır yük altındayken veya başarısız olduğunda zincirleme hataları önler.
-   **Mekanizma**: Hataları izler. Bir eşiğe ulaşılırsa, devreyi "açar" (open) ve istekleri bir zaman aşımı süresi boyunca hemen reddeder. Ardından, servisin düzelip düzelmediğini test etmek için "yarı açık" (half-open) durumuna geçer.

### 2. Sağlık İzleme (Health Monitoring) - `health.go`
-   **Amaç**: Sistem sağlık ölçümlerini toplar ve sunar.
-   **Ölçümler**: CPU kullanımı, RAM kullanımı, Disk kullanımı, Ağ istatistikleri, Goroutine sayısı, DB istatistikleri ve Çalışma Süresi (Uptime).

### 3. Metrikler (Metrics) - `metrics.go`
-   **Amaç**: Uygulama seviyesindeki metrikleri izler.
-   **Ölçümler**: Toplam istekler, Hata oranları (4xx/5xx), Yanıt süreleri ve HTTP metoduna göre istek sayıları.

### 4. Kimlik Doğrulama (Authentication) - `middlewares.go`
-   **`AuthMiddleware`**: `Auth` çerezindeki JWT token'ını doğrular. Ayrıca korumalı rotalar için kullanıcının "admin" rolüne sahip olup olmadığını kontrol eder.
-   **`TimeoutMiddleware`**: Uzun süren bağlantıları önlemek için isteklere bir zaman aşımı (timeout) süresi belirler.
-   **`RequestLoggerMiddleWare`**: Kabul edilen her HTTP isteğinin detaylarını loglar (Metot, Yol, IP, Durum, Süre).

### 5. Hız Sınırlayıcı (Rate Limiter) - `ratelimiter.go`
-   **Amaç**: Bir istemcinin (IP) belirli bir zaman aralığında yapabileceği istek sayısını sınırlayarak kötüye kullanımı önler.
-   **Uygulama**: IP adresi başına token kovası algoritması.

---

## Veri Modelleri (Models)

### 1. Users (Kullanıcılar)
-   **Amaç**: Sistem yöneticilerini ve kullanıcıları yönetir.
-   **Önemli Alanlar**: `Username` (Kullanıcı Adı), `Password` (Hashlenmiş Şifre), `Role` (admin/user).

### 2. Facilitators (Konuşmacılar/Kolaylaştırıcılar)
-   **Amaç**: Konuşmacıları veya atölye liderlerini temsil eder.
-   **Önemli Alanlar**: `Name`, `Title`, `Topic`, `TopicDetails`, `Photograph`.

### 3. Sponsors (Sponsorlar)
-   **Amaç**: Etkinlik sponsorlarını yönetir.
-   **Önemli Alanlar**: `SponsorName`, `SponsorTier` (Altın, Gümüş vb.), `Logo`, `AdvertiseVideo`.

### 4. Workshops & TimeSlots (Atölyeler ve Zaman Dilimleri)
-   **`Workshops`**: Bir ana etkinliği veya atölye oturumunu temsil eder.
    -   **Önemli Alanlar**: `WorkshopName`, `WorkshopDate`, `IsLive`.
-   **`WorkshopTimeSlot`**: Bir atölye içindeki belirli bir zaman dilimi.
    -   **Önemli Alanlar**: `WorkshopID`, `FaciliatorID`, `SlotStart`, `SlotEnd`, `SlotOrder`.

---

## API Uç Noktaları (Controllers)

Aşağıda API uç noktaları ve beklenen JSON istek gövdeleri (request body) listelenmiştir.

### Kimlik Doğrulama & Kullanıcılar (`usercontroller.go`)

| Metot | Uç Nokta | Açıklama |
| :--- | :--- | :--- |
| **POST** | `/signup` | Yeni bir kullanıcı hesabı oluşturur. |
| **POST** | `/login` | Giriş yapar ve JWT çerezi alır. |
| **GET** | `/users` | Tüm kullanıcıları listeler. |
| **DELETE** | `/users/:id` | Belirli bir kullanıcıyı siler. |

**Örnek JSON (Signup):**
```json
{
  "username": "kullanici1",
  "password": "sifre123",
  "role": "user"
}
```

**Örnek JSON (Login):**
```json
{
  "username": "kullanici1",
  "password": "sifre123"
}
```

### Konuşmacılar (`faciliatorcontroller.go`)

| Metot | Uç Nokta | Açıklama |
| :--- | :--- | :--- |
| **POST** | `/facilitators` | Yeni bir konuşmacı oluşturur. |
| **GET** | `/facilitators` | Tüm konuşmacıları listeler. |
| **GET** | `/facilitators/topic/:topic` | Konuya göre konuşmacıları getirir. |
| **PUT** | `/facilitators/:id` | Konuşmacı detaylarını günceller. |
| **DELETE** | `/facilitators/:id` | Bir konuşmacıyı siler. |

**Örnek JSON (Create/Update):**
```json
{
  "name": "Ahmet Yılmaz",
  "title": "GDE",
  "topic": "Android Geliştirme",
  "topic_details": "Jetpack Compose Detayları",
  "photograph": "/images/ahmet.jpg"
}
```

### Sponsorlar (`sponsorcontroller.go`)

| Metot | Uç Nokta | Açıklama |
| :--- | :--- | :--- |
| **POST** | `/sponsors` | Yeni bir sponsor ekler. |
| **GET** | `/sponsors` | Tüm sponsorları listeler. |
| **DELETE** | `/sponsors/:id` | Bir sponsoru siler. |

**Örnek JSON (Create):**
```json
{
  "sponsor_name": "Google",
  "sponsor_tier": "Gold",
  "logo": "/logos/google.png",
  "advertise_video": "/videos/google_reklam.mp4",
  "website": "https://google.com"
}
```

### Atölyeler & Program (`workshopcontroller.go`)

| Metot | Uç Nokta | Açıklama |
| :--- | :--- | :--- |
| **GET** | `/workshops` | Tüm atölyeleri listeler. |
| **POST** | `/workshops` | Yeni atölye oluşturur (slot'larla birlikte olabilir). |
| **PUT** | `/workshops/:id` | Atölye detaylarını günceller. |
| **DELETE** | `/workshops/:id` | Atölyeyi ve slotlarını siler. |
| **GET** | `/workshops/:id/schedule` | Bir atölyenin tam programını getirir. |
| **POST** | `/workshops/:id/slots` | Mevcut bir atölyeye zaman dilimi ekler. |
| **DELETE** | `/workshops/slots/:id` | Belirli bir zaman dilimini siler. |
| **PUT** | `/workshops/slots/:id` | Belirli bir zaman dilimini günceller. |
| **POST** | `/workshops/:id/delay` | Atölyeye gecikme (dakika) ekler/çıkarır. |
| **POST** | `/workshops/:id/live` | Atölyenin "Canlı" durumunu değiştirir. |

**Örnek JSON (Create Workshop):**
```json
{
  "workshop_name": "Go Programlama Kampı",
  "workshop_date": "2023-10-27T09:00:00Z",
  "time_slots": [
    {
       "faciliator_id": 1,
       "slot_start": "2023-10-27T09:00:00Z",
       "slot_end": "2023-10-27T10:00:00Z"
    }
  ]
}
```

**Örnek JSON (Add Slots):**
```json
{
  "time_slots": [
    {
       "faciliator_id": 2,
       "slot_start": "2023-10-27T10:00:00Z",
       "slot_end": "2023-10-27T11:00:00Z"
    }
  ]
}
```

**Örnek JSON (Update Workshop):**
```json
{
  "workshop_name": "Golang İleri Seviye"
}
```

**Örnek JSON (Update Slot):**
```json
{
  "slot_start": "2023-10-27T10:15:00Z"
}
```

**Örnek JSON (Add Delay):**
```json
{
  "delay_minutes": 15
}
```
*(Negatif değerler programı erkene çeker)*

**Örnek JSON (Set Live):**
```json
{
  "is_live": true
}
```

### WebSockets (`websockets.go`)

Gerçek zamanlı veri akışları sağlar.

| Uç Nokta | Açıklama |
| :--- | :--- |
| `/ws/slots/current` | Şu anda aktif olan slotların akışı. |
| `/ws/slots/upcoming` | Yaklaşan slotların akışı (sıradaki 5). |
| `/ws/sponsors` | Sponsor verilerinin akışı. |
| `/ws/workshops/:id/schedule` | Belirli bir atölyenin tam program akışı (değişiklik olduğunda güncellenir). |
| `/ws/workshops/:id/current` | Belirli bir atölyenin anlık durumu ve canlı slot akışı. |