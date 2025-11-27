package models

import "time"

type Faciliators struct {
	FaciliatorID uint      `json:"faciliator_id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null"`
	Title        string    `json:"title" gorm:"type:varchar(100);not null"` // GDE, Android Expert vb.
	Topic        string    `json:"topic" gorm:"type:varchar(200);not null"` //
	TopicDetails string    `json:"topic_details" gorm:"type:text;not null"` //
	Photograph   string    `json:"photograph" gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
