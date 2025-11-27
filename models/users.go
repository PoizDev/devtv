package models

import "time"

type User struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"type:varchar(50);unique;not null"`
	Password  string    `json:"password" gorm:"not null"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
