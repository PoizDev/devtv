package models

type Sponsors struct {
	SponsorID      uint   `json:"sponsor_id" gorm:"primaryKey;autoIncrement"`
	SponsorName    string `json:"sponsor_name" gorm:"varchar(100);not null;unique;"`
	Logo           string `json:"logo" gorm:"type:varchar(255);not null;"`   //Logo URL / file path
	AdvertiseVideo string `json:"advertise_video" gorm:"type:varchar(255);"` // Reklam video URL / file path
	Website        string `json:"website" gorm:"type:varchar(255);"`
}
