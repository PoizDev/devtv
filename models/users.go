package models

import "time"

type User struct {
	ID        uint      `json:"user_id" gorm:"column:user_id;primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"type:varchar(50);unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:'user'"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	SurveyResponses []UserSurveyResponse `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
}
