package models

import "time"

// Workshops - Ana workshop bilgisi
type Workshops struct {
	WorkshopID   uint      `json:"workshop_id" gorm:"primaryKey;autoIncrement"`
	WorkshopName string    `json:"workshop_name" gorm:"type:varchar(100);not null"` // "Çam Atölyesi"
	WorkshopDate time.Time `json:"workshop_date" gorm:"type:date;not null"`
	IsLive       bool      `json:"is_live" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// İlişkiler
	TimeSlots []WorkshopTimeSlot `json:"time_slots" gorm:"foreignKey:WorkshopID"`
}

// WorkshopTimeSlot - Her zaman dilimi için ayrı kayıt
type WorkshopTimeSlot struct {
	SlotID     uint       `json:"slot_id" gorm:"primaryKey;autoIncrement"`
	WorkshopID uint       `json:"workshop_id" gorm:"not null;index"` // Hangi workshop? (Çam=1, Fidan=2)
	Workshop   *Workshops `json:"workshop,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	FaciliatorID uint         `json:"faciliator_id" gorm:"not null;index"`
	Faciliator   *Faciliators `json:"faciliator,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	SlotStart time.Time `json:"slot_start" gorm:"type:timestamp;not null;index"`
	SlotEnd   time.Time `json:"slot_end" gorm:"type:timestamp;not null"`
	SlotOrder int       `json:"slot_order" gorm:"not null"` // Sıralama: 1, 2, 3...

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DTO'lar - API Response için
type WorkshopScheduleResponse struct {
	WorkshopID   uint               `json:"workshop_id"`
	WorkshopName string             `json:"workshop_name"`
	WorkshopDate time.Time          `json:"workshop_date"`
	IsLive       bool               `json:"is_live"`
	CurrentSlot  *TimeSlotResponse  `json:"current_slot,omitempty"`
	AllSlots     []TimeSlotResponse `json:"all_slots"`
	TotalSlots   int                `json:"total_slots"`
}

type TimeSlotResponse struct {
	SlotID     uint               `json:"slot_id"`
	SlotStart  time.Time          `json:"slot_start"`
	SlotEnd    time.Time          `json:"slot_end"`
	SlotOrder  int                `json:"slot_order"`
	Faciliator FaciliatorResponse `json:"faciliator"`
}

type FaciliatorResponse struct {
	FaciliatorID uint   `json:"faciliator_id"`
	Name         string `json:"name"`
	Topic        string `json:"topic"`
	TopicDetails string `json:"topic_details"`
	Photograph   string `json:"photograph"`
}

type UpcomingSlotResponse struct {
	SlotID         uint               `json:"slot_id"`
	WorkshopName   string             `json:"workshop_name"`
	SlotStart      time.Time          `json:"slot_start"`
	SlotEnd        time.Time          `json:"slot_end"`
	Faciliator     FaciliatorResponse `json:"faciliator"`
	TimeUntilStart string             `json:"time_until_start"` // "15 dakika sonra"
}
