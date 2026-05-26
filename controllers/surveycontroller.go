package controllers

import (
	"context"
	"devtv/config"
	"devtv/in"
	"devtv/models"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

const activeQuestionsCacheKey = "survey:active_questions"

func GetActiveQuestions(c *gin.Context) {
	if in.RDB != nil {
		ctx := context.Background()
		cached, err := in.RDB.Get(ctx, activeQuestionsCacheKey).Result()
		if err == nil {
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}
	}

	var questions []models.SurveyQuestion
	err := in.DB.Preload("Options").Where("is_active = ?", true).Order("\"order\" asc").Find(&questions).Error
	if err != nil {
		config.Log.Error("Aktif sorular alınamadı", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sorular alınırken bir hata oluştu"})
		return
	}

	if in.RDB != nil {
		ctx := context.Background()
		if jsonBytes, err := json.Marshal(questions); err == nil {
			in.RDB.Set(ctx, activeQuestionsCacheKey, jsonBytes, 15*time.Minute)
		}
	}

	c.JSON(http.StatusOK, questions)
}

type SurveySubmitRequest struct {
	Answers []struct {
		QuestionID uint `json:"question_id" binding:"required"`
		OptionID   uint `json:"option_id" binding:"required"`
	} `json:"answers" binding:"required"`
}

func SubmitSurvey(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kullanıcı bilgisi bulunamadı"})
		return
	}

	var req SurveySubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek formatı"})
		return
	}

	var responses []models.UserSurveyResponse
	for _, ans := range req.Answers {
		responses = append(responses, models.UserSurveyResponse{
			UserID:     userID,
			QuestionID: ans.QuestionID,
			OptionID:   ans.OptionID,
		})
	}

	//' Kullanıcı aynı soruya tekrar cevap verirse çakışmayı engellemek için Upsert yapıyoruz
	err := in.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "question_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"option_id", "updated_at"}),
	}).Create(&responses).Error

	if err != nil {
		config.Log.Error("Kullanıcı cevapları kaydedilemedi", zap.Error(err), zap.Uint("userID", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cevaplarınız kaydedilirken bir hata oluştu"})
		return
	}

	results, err := calculateSurveyResults(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonuçlar hesaplanırken bir hata oluştu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Anket başarıyla kaydedildi",
		"results": results,
	})
}

func GetSurveyResults(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kullanıcı bilgisi bulunamadı"})
		return
	}

	results, err := calculateSurveyResults(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonuçlar alınırken bir hata oluştu"})
		return
	}

	c.JSON(http.StatusOK, results)
}

type ScheduleSlotResponse struct {
	models.TimeSlotResponse
	WorkshopName string `json:"workshop_name"`
	MatchScore   int    `json:"match_score"`
}

type SurveyCalculationResult struct {
	TopCategories       []Score                `json:"top_categories"`
	TopTags             []Score                `json:"top_tags"`
	RecommendedSchedule []ScheduleSlotResponse `json:"recommended_schedule"`
}

type Score struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

const upcomingSlotsCacheKey = "schedule:upcoming_slots"

func getUpcomingSlots() ([]models.WorkshopTimeSlot, error) {
	var slots []models.WorkshopTimeSlot
	ctx := context.Background()

	if in.RDB != nil {
		cached, err := in.RDB.Get(ctx, upcomingSlotsCacheKey).Result()
		if err == nil {
			if unmarshalErr := json.Unmarshal([]byte(cached), &slots); unmarshalErr == nil {
				return slots, nil
			}
		}
	}

	now := time.Now()
	err := in.DB.Preload("Facilitator").Preload("Facilitator.Tags").Preload("Workshop").
		Where("slot_start > ?", now).
		Find(&slots).Error

	if err != nil {
		return nil, err
	}

	if in.RDB != nil {
		if jsonBytes, err := json.Marshal(slots); err == nil {
			in.RDB.Set(ctx, upcomingSlotsCacheKey, jsonBytes, 30*time.Minute)
		}
	}

	return slots, nil
}

func calculateSurveyResults(userID uint) (*SurveyCalculationResult, error) {
	var responses []models.UserSurveyResponse
	err := in.DB.Preload("Option.Tag.Categories").Where("user_id = ?", userID).Find(&responses).Error
	if err != nil {
		config.Log.Error("Kullanıcı cevapları okunurken hata oluştu", zap.Error(err))
		return nil, err
	}

	tagScores := make(map[string]int)
	categoryScores := make(map[string]int)

	for _, resp := range responses {
		opt := resp.Option
		tag := opt.Tag
		if tag.ID != 0 {
			tagScores[tag.Name] += opt.Points
			for _, cat := range tag.Categories {
				if cat.ID != 0 {
					categoryScores[cat.Name] += opt.Points
				}
			}
		}
	}

	topCats := sortMapToSlice(categoryScores)
	topTags := sortMapToSlice(tagScores)

	//' Anket sonuçlarına göre kullanıcıya en uygun olan Akıllı Takvimi (Smart Scheduling) oluşturuyoruz
	upcomingSlots, err := getUpcomingSlots()
	if err != nil {
		config.Log.Error("Yaklaşan slotlar alınamadı", zap.Error(err))
	}

	type SlotWithScore struct {
		Slot  models.WorkshopTimeSlot
		Score int
	}

	var scoredSlots []SlotWithScore
	for _, slot := range upcomingSlots {
		if slot.Facilitator == nil {
			continue
		}
		score := 0
		for _, ft := range slot.Facilitator.Tags {
			score += tagScores[ft.Name]
		}
		if score > 0 {
			scoredSlots = append(scoredSlots, SlotWithScore{Slot: slot, Score: score})
		}
	}

	//' Sıralama: Önce Başlangıç Zamanı (ASC), sonra Uyum Skoru (DESC)
	sort.Slice(scoredSlots, func(i, j int) bool {
		if scoredSlots[i].Slot.SlotStart.Equal(scoredSlots[j].Slot.SlotStart) {
			return scoredSlots[i].Score > scoredSlots[j].Score
		}
		return scoredSlots[i].Slot.SlotStart.Before(scoredSlots[j].Slot.SlotStart)
	})

	var recommendedSchedule []ScheduleSlotResponse
	var lastEndTime time.Time

	for _, ss := range scoredSlots {
		if ss.Slot.SlotStart.Before(lastEndTime) {
			//' Oturum çakışması tespit edildiği için bu oturumu atlıyoruz
			continue
		}

		recommendedSchedule = append(recommendedSchedule, ScheduleSlotResponse{
			TimeSlotResponse: models.TimeSlotResponse{
				SlotID:    ss.Slot.SlotID,
				SlotStart: ss.Slot.SlotStart,
				SlotEnd:   ss.Slot.SlotEnd,
				SlotOrder: ss.Slot.SlotOrder,
				Facilitator: models.FacilitatorResponse{
					FacilitatorID: ss.Slot.Facilitator.FacilitatorID,
					Name:          ss.Slot.Facilitator.Name,
					Topic:         ss.Slot.Facilitator.Topic,
					Tags:          ss.Slot.Facilitator.Tags,
					TopicDetails:  ss.Slot.Facilitator.TopicDetails,
					Photograph:    ss.Slot.Facilitator.Photograph,
				},
			},
			WorkshopName: ss.Slot.Workshop.WorkshopName,
			MatchScore:   ss.Score,
		})
		lastEndTime = ss.Slot.SlotEnd
	}

	return &SurveyCalculationResult{
		TopCategories:       topCats,
		TopTags:             topTags,
		RecommendedSchedule: recommendedSchedule,
	}, nil
}

func sortMapToSlice(m map[string]int) []Score {
	var scores []Score
	for k, v := range m {
		if v > 0 {
			scores = append(scores, Score{Name: k, Score: v})
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].Name < scores[j].Name
		}
		return scores[i].Score > scores[j].Score
	})

	return scores
}
