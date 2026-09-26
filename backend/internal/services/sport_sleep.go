package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type SportService struct {
	db *sql.DB
}

func NewSportService(db *sql.DB) *SportService {
	return &SportService{db: db}
}

func (s *SportService) CreateSession(userID int64, session *models.SportSession) error {
	session.UserID = userID
	session.CompletedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO sport_sessions (user_id, workout_type, completed, duration_minutes, metric_value, metric_type, notes, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.UserID, session.WorkoutType, session.Completed, session.DurationMinutes, session.MetricValue, session.MetricType, session.Notes, session.CompletedAt,
	)
	return err
}

func (s *SportService) GetTodayDone(userID int64) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sport_sessions WHERE user_id = ? AND completed = 1 AND date(completed_at) = date('now')`,
		userID,
	).Scan(&count)
	return count > 0, err
}

func (s *SportService) GetSessions(userID int64, limit int) ([]models.SportSession, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, workout_type, completed, duration_minutes, metric_value, metric_type, notes, completed_at FROM sport_sessions WHERE user_id = ? ORDER BY completed_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.SportSession
	for rows.Next() {
		var sess models.SportSession
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.WorkoutType, &sess.Completed, &sess.DurationMinutes, &sess.MetricValue, &sess.MetricType, &sess.Notes, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

type SleepService struct {
	db *sql.DB
}

func NewSleepService(db *sql.DB) *SleepService {
	return &SleepService{db: db}
}

func (s *SleepService) CreateRecord(userID int64, record *models.SleepRecord) error {
	record.UserID = userID
	record.RecordedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO sleep_records (user_id, sleep_hours, bedtime, wake_time, recorded_at) VALUES (?, ?, ?, ?, ?)`,
		record.UserID, record.SleepHours, record.Bedtime, record.WakeTime, record.RecordedAt,
	)
	return err
}

func (s *SleepService) GetTodayHours(userID int64) (*float64, error) {
	var hours float64
	err := s.db.QueryRow(
		`SELECT sleep_hours FROM sleep_records WHERE user_id = ? AND date(recorded_at) = date('now') ORDER BY recorded_at DESC LIMIT 1`,
		userID,
	).Scan(&hours)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &hours, err
}

func (s *SleepService) GetRecords(userID int64, limit int) ([]models.SleepRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, sleep_hours, bedtime, wake_time, recorded_at FROM sleep_records WHERE user_id = ? ORDER BY recorded_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.SleepRecord
	for rows.Next() {
		var rec models.SleepRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.SleepHours, &rec.Bedtime, &rec.WakeTime, &rec.RecordedAt); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}