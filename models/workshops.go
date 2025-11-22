package models

import "time"

type Workshops struct {
	WorkshopID      uint         `json:"workshop_id" gorm:"primaryKey;autoIncrement"`
	WorkshopName    string       `json:"workshop_name" gorm:"type:varchar(100);not null"`
	FaciliatorID    uint         `json:"faciliator_id"`
	Faciliator      *Faciliators `json:"faciliator" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	WorkshopDetails string       `json:"workshop_details" gorm:"type:text;not null"`
	WorkshopStart   time.Time    `json:"workshop_start" gorm:"type:timestamp;not null"`
	WorkshopEnd     time.Time    `json:"workshop_end" gorm:"type:timestamp;not null"`
	IsLive          bool         `json:"is_live" gorm:"type:boolean;not null;default:false"`
}
