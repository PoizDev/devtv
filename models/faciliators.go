package models

type Faciliators struct {
	FaciliatorID uint   `json:"faciliator_id" gorm:"primaryKey;autoIncrement"`
	Name         string `json:"name" gorm:"type:varchar(100);not null"`
	Topic        string `json:"topic" gorm:"type:varchar(100);not null"`
	Photoragph   string `json:"photograph" gorm:"type:varchar(255);not null"`
}
