package models

import "time"

type PlayerProfile struct {
	ID              int       `json:"id"`
	Username        string    `json:"username"`
	Avatar          string    `json:"avatar"`
	Balance         int       `json:"balance"`           // Текущее золото
	Wins            int       `json:"wins"`              // Количество побед
	Games           int       `json:"games"`             // Всего сыграно партий
	LastWeeklyClaim time.Time `json:"last_weekly_claim"` // время выдачи еженедельного золота
}
