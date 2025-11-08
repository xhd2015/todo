package models

import "time"

type LearningMaterial struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Content           string    `json:"content"`
	Source            string    `json:"source"`
	Type              string    `json:"type"`
	Difficulty        string    `json:"difficulty"`
	CreateTime        time.Time `json:"create_time"`
	UpdateTime        time.Time `json:"update_time"`
	LastViewBeginTime time.Time `json:"last_view_begin_time"`
	LastViewEndTime   time.Time `json:"last_view_end_time"`
	Count             int64     `json:"count"`
}
