package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type RadarService struct {
	db *sql.DB
}

func NewRadarService(db *sql.DB) *RadarService {
	return &RadarService{db: db}
}

func (s *RadarService) CalculateAndStore(userID int64) (*models.RadarScores, error) {
	studyScore := s.calcStudyScore(userID)
	sportScore := s.calcSportScore(userID)
	hobbyScore := s.calcHobbyScore(userID)
	routineScore := s.calcRoutineScore(userID)

	scores := &models.RadarScores{
		UserID:        userID,
		StudyScore:    studyScore,
		SportScore:    sportScore,
		HobbyScore:    hobbyScore,
		RoutineScore:  routineScore,
		CalculatedAt:  time.Now(),
	}

	_, err := s.db.Exec(
		`INSERT INTO radar_scores (user_id, study_score, sport_score, hobby_score, routine_score, calculated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		scores.UserID, scores.StudyScore, scores.SportScore, scores.HobbyScore, scores.RoutineScore, scores.CalculatedAt,
	)
	if err != nil {
		return nil, err
	}

	return scores, nil
}

func (s *RadarService) GetLatest(userID int64) (*models.RadarScores, error) {
	var scores models.RadarScores
	err := s.db.QueryRow(
		`SELECT id, user_id, study_score, sport_score, hobby_score, routine_score, calculated_at FROM radar_scores WHERE user_id = ? ORDER BY calculated_at DESC LIMIT 1`,
		userID,
	).Scan(&scores.ID, &scores.UserID, &scores.StudyScore, &scores.SportScore, &scores.HobbyScore, &scores.RoutineScore, &scores.CalculatedAt)
	if err == sql.ErrNoRows {
		return s.CalculateAndStore(userID)
	}
	return &scores, err
}

func (s *RadarService) calcStudyScore(userID int64) float64 {
	var totalMinutes int
	s.db.QueryRow(
		`SELECT COALESCE(SUM(duration_minutes), 0) FROM study_sessions WHERE user_id = ? AND completed_at >= date('now', '-7 days')`,
		userID,
	).Scan(&totalMinutes)

	targetMinutes := 420.0 // 7 hours per week target
	score := (float64(totalMinutes) / targetMinutes) * 100
	if score > 100 {
		score = 100
	}
	return score
}

func (s *RadarService) calcSportScore(userID int64) float64 {
	var daysActive int
	var totalMinutes int
	s.db.QueryRow(
		`SELECT COUNT(DISTINCT date(completed_at)) FROM sport_sessions WHERE user_id = ? AND completed = 1 AND completed_at >= date('now', '-7 days')`,
		userID,
	).Scan(&daysActive)
	s.db.QueryRow(
		`SELECT COALESCE(SUM(duration_minutes), 0) FROM sport_sessions WHERE user_id = ? AND completed = 1 AND completed_at >= date('now', '-7 days')`,
		userID,
	).Scan(&totalMinutes)

	targetDays := 4.0
	targetMinutes := 180.0
	score := ((float64(daysActive)/targetDays)*0.4 + (float64(totalMinutes)/targetMinutes)*0.6) * 100
	if score > 100 {
		score = 100
	}
	return score
}

func (s *RadarService) calcHobbyScore(userID int64) float64 {
	var totalSessions int
	s.db.QueryRow(
		`SELECT COUNT(*) FROM hobby_sessions WHERE user_id = ? AND completed_at >= date('now', '-7 days')`,
		userID,
	).Scan(&totalSessions)

	targetSessions := 7.0 // daily target
	score := (float64(totalSessions) / targetSessions) * 100
	if score > 100 {
		score = 100
	}
	return score
}

func (s *RadarService) calcRoutineScore(userID int64) float64 {
	var sleepDays int
	s.db.QueryRow(
		`SELECT COUNT(DISTINCT date(recorded_at)) FROM sleep_records WHERE user_id = ? AND sleep_hours >= 7 AND recorded_at >= date('now', '-7 days')`,
		userID,
	).Scan(&sleepDays)

	targetDays := 7.0
	score := (float64(sleepDays) / targetDays) * 100
	if score > 100 {
		score = 100
	}
	return score
}

func (s *RadarService) GetDashboardData(userID int64) (*models.DashboardData, error) {
	radar, err := s.GetLatest(userID)
	if err != nil {
		return nil, err
	}

	studySvc := NewStudyService(s.db)
	sportSvc := NewSportService(s.db)
	sleepSvc := NewSleepService(s.db)
	hobbySvc := NewHobbyService(s.db)
	f1Svc := NewF1Service(s.db)
	mediaSvc := NewMediaService(s.db)

	todayStudy, _ := studySvc.GetTodayMinutes(userID)
	todaySport, _ := sportSvc.GetTodayDone(userID)
	todaySleep, _ := sleepSvc.GetTodayHours(userID)
	streaks, _ := hobbySvc.GetStreaks(userID)
	if streaks == nil {
		streaks = []models.HobbyStreak{}
	}

	f1Events, _ := f1Svc.GetUpcoming(userID, 3)
	if f1Events == nil {
		f1Events = []models.F1Event{}
	}

	recentMedia, _ := mediaSvc.GetRecent(userID, 3)
	if recentMedia == nil {
		recentMedia = []models.MediaItem{}
	}

	activeStreaks := make([]models.HobbyStreak, 0, len(streaks))
	for _, st := range streaks {
		if st.CurrentStreak > 0 {
			activeStreaks = append(activeStreaks, st)
		}
	}

	return &models.DashboardData{
		RadarScores:   *radar,
		TodayStudy:    todayStudy,
		TodaySport:    todaySport,
		TodaySleep:    todaySleep,
		ActiveStreaks: activeStreaks,
		UpcomingF1:    f1Events,
		RecentMedia:   recentMedia,
	}, nil
}