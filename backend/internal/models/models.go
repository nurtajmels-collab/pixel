package models

import "time"

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id,omitempty"`
	Username   string    `json:"username,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type StudySession struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Category        string    `json:"category"`
	DurationMinutes int       `json:"duration_minutes"`
	QualityRating   *int      `json:"quality_rating,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
}

type StudyScore struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Category   string    `json:"category"`
	Subject    string    `json:"subject,omitempty"`
	Component  string    `json:"component,omitempty"`
	Score      float64   `json:"score"`
	MaxScore   float64   `json:"max_score,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

type StudyAssessment struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Subject        string    `json:"subject"`
	AssessmentType string    `json:"assessment_type"`
	Score          float64   `json:"score"`
	MaxScore       float64   `json:"max_score"`
	Notes          string    `json:"notes,omitempty"`
	RecordedAt     time.Time `json:"recorded_at"`
}

type StudySubject struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Name      string  `json:"name"`
	FAWeight  float64 `json:"fa_weight"`
	SAUWeight float64 `json:"sau_weight"`
	SATWeight float64 `json:"sat_weight"`
}

type MediaItem struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	Type           string     `json:"type"`
	Title          string     `json:"title"`
	CoverURL       string     `json:"cover_url,omitempty"`
	Status         string     `json:"status"`
	CurrentSeason  *int       `json:"current_season,omitempty"`
	CurrentEpisode *int       `json:"current_episode,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type SportSession struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	WorkoutType  string    `json:"workout_type,omitempty"`
	Completed    bool      `json:"completed"`
	DurationMinutes int    `json:"duration_minutes,omitempty"`
	MetricValue  *float64  `json:"metric_value,omitempty"`
	MetricType   string    `json:"metric_type,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

type SleepRecord struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	SleepHours float64  `json:"sleep_hours"`
	Bedtime   string    `json:"bedtime,omitempty"`
	WakeTime  string    `json:"wake_time,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

type HobbyStreak struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	Name            string     `json:"name"`
	CurrentStreak   int        `json:"current_streak"`
	LongestStreak   int        `json:"longest_streak"`
	LastCompletedAt *time.Time `json:"last_completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type HobbySession struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	HobbyStreakID    *int64    `json:"hobby_streak_id,omitempty"`
	DurationMinutes  *int      `json:"duration_minutes,omitempty"`
	MetricValue      *float64  `json:"metric_value,omitempty"`
	MetricType       string    `json:"metric_type,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	CompletedAt      time.Time `json:"completed_at"`
}

type F1Event struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	EventDate string    `json:"event_date"`
	EventType string    `json:"event_type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RadarScores struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	StudyScore    float64   `json:"study_score"`
	SportScore    float64   `json:"sport_score"`
	HobbyScore    float64   `json:"hobby_score"`
	RoutineScore  float64   `json:"routine_score"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

type DashboardData struct {
	RadarScores   RadarScores    `json:"radar_scores"`
	TodayStudy    int            `json:"today_study_minutes"`
	TodaySport    bool           `json:"today_sport_done"`
	TodaySleep    *float64       `json:"today_sleep_hours,omitempty"`
	ActiveStreaks []HobbyStreak  `json:"active_streaks"`
	UpcomingF1    []F1Event      `json:"upcoming_f1"`
	RecentMedia   []MediaItem    `json:"recent_media"`
}

const (
	MediaTypeBook    = "book"
	MediaTypeSeries  = "series"
	MediaStatusPlanned   = "planned"
	MediaStatusInProgress = "in_progress"
	MediaStatusDone      = "done"
)

const (
	StudyCategorySAT   = "SAT"
	StudyCategoryIELTS = "IELTS"
	StudyCategoryNISH  = "NISH"
)