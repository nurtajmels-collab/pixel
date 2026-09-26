package models

import "time"

type Activity struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	StartTime     *string   `json:"start_time,omitempty"`
	EndTime       *string   `json:"end_time,omitempty"`
	DurationHours *float64  `json:"duration_hours,omitempty"`
	Tags          string    `json:"tags"`
	CreatedAt     time.Time `json:"created_at"`
}

type Reminder struct {
	ID       int64     `json:"id"`
	UserID   int64     `json:"user_id"`
	Task     string    `json:"task"`
	RemindAt time.Time `json:"remind_at"`
	IsSent   bool      `json:"is_sent"`
}
