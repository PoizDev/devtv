package models

import "time"

type SurveyQuestion struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Text      string    `json:"text" gorm:"type:text;not null"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	Order     int       `json:"order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Options []SurveyOption `json:"options" gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE;"`
}

type SurveyOption struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	QuestionID uint      `json:"question_id" gorm:"not null"`
	Text       string    `json:"text" gorm:"type:varchar(255);not null"`
	TagID      uint      `json:"tag_id" gorm:"not null"`
	Points     int       `json:"points" gorm:"default:1"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`

	Tag Tag `json:"tag" gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE;"`
}

type UserSurveyResponse struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_question"`
	QuestionID uint      `json:"question_id" gorm:"not null;uniqueIndex:idx_user_question"`
	OptionID   uint      `json:"option_id" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Question SurveyQuestion `json:"-" gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE;"`
	Option   SurveyOption   `json:"-" gorm:"foreignKey:OptionID;constraint:OnDelete:CASCADE;"`
}
