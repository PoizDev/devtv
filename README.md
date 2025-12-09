# GDG Bursa - DevTV

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![Gin Framework](https://img.shields.io/badge/Gin-v1.11.0-00ADD8?style=for-the-badge&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-316192?style=for-the-badge&logo=postgresql)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)

## Genel Bakış

Bu sistem Devfest Bursa 2025 için geliştirilmiş bir etkinlik akışı sistemidir. Bu sistem production'a hazırlık açısından pek çok önlem ve özellikle bezenmiştir.

## Kullanılan Teknolojiler
Dil: Golang Version 1.24.5
Framework: Gin Web Framework
DB: PostgreSQL 15+
ORM: GORM v2

## Sistem Kurulumu

1. Repoyu Klonlayın 
```
    git clone https://github.com/poizdev/devtv.git
```
-eğer bu aşamada bi sorunla karşılaşırsanız github üzerinden klasöre indir yapabilirsiniz-

2. Bağımlılıkları Yükleyin

```
    go mod download
    go mod tidy
```

3. in/devtv.env dosyasını oluşturun (örnek ektedir.)

```
    dsn="user=kullaniciadi password=sifreniz dbname=dbadi port=5432 sslmode=disable TimeZone=Europe/Istanbul"
    JWT_SECRET="gizli keyiniz"
```

4. Uygulamayı çalıştırın
```
    go run main.go
```

## API Dokümantasyonu

### Base URL:
```
http://localhost:2012
```

### Kimlik Doğrulama ve Admin girişi

DevTV sistemi, arka planda JWT Token ve Role Auth sistemlerini kullanır. Sistemi ilk açtığınızda bir Admin user'ı oluşturmanız ve o token'ı kullanmanız gerekmektedir. /admin ile korunan endpointlere o token'ı kullanarak istek gönderebilirsiniz. Bunun için örnek istek:

```
POST - localhost:2012/signup

{
    "username":"admin",
    "password":"çokgüçlüşifre",
    "role":"admin
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
        "title": "Örnek Title",
        "topic": "Örnek Konu",
        "topic_details": "Örnek Konu Detayları",
        "photograph": "/public/faciliators/ornek.png",
        "created_at": "2025-12-08T23:18:14.266189+03:00",
        "updated_at": "2025-12-08T23:18:14.266189+03:00"
    }
]
```
photograph parametresi string bir biçimde dosya yolunu veriyor. Bunun için devtv'nin frontunda önlemler alındı. dosya girişi de buna göre olmalı.

2. Facilitator oluşturma
**!!Önemli: Bu istek ve diğer başı bütün /admin olan istekler JWT Token gerektirmektedir. Önce Login'den Auth tokenleri alın, sonra eğer postmande çalışıyorsanız Header kısmında**
```
Key:Cookie
Value: 
Auth=<jwt-token>; Path=/; Domain=localhost; Max-Age=2592000; HttpOnly; SameSite=None
```
**olmalıdır**

```
POST - localhost:2012/admin/create/faciliator
Örnek JSON Body:  

{
    "name":"Örnek Konuşmacı",
    "title":"Örnek Title",
    "topic":"Örnek Konu",
    "topic_details":"Örnek Konu Detayları",
    "photograph":"/public/faciliators/ornek.png"
}
```

3. Facilitator silme
```
(:id kısmına silmek istediğimiz ID'nin inputu verilecektir)
DELETE - localhost:2012/admin/faciliator/:id
```

4. Facilitator güncelleme
```
Not: Bütün update parametrelerinde sadece güncellemek istediğiniz parametreyi ve değeri yazmanız yeterli olur.
PUT - localhost:2012/admin/faciliator/:id

JSON Body Örneği:
{
    "name":"Emre Hızlı",
    "photograph":"/public/faciliators/emrehizli.jpeg"
}
```

