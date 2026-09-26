package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type HobbyService struct {
	db *sql.DB
}

func NewHobbyService(db *sql.DB) *HobbyService {
	return &HobbyService{db: db}
}

func (s *HobbyService) CreateStreak(userID int64, streak *models.HobbyStreak) error {
	streak.UserID = userID
	streak.CreatedAt = time.Now()
	result, err := s.db.Exec(
		`INSERT INTO hobby_streaks (user_id, name, current_streak, longest_streak, last_completed_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		streak.UserID, streak.Name, streak.CurrentStreak, streak.LongestStreak, streak.LastCompletedAt, streak.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	streak.ID = id
	return nil
}

func (s *HobbyService) GetStreaks(userID int64) ([]models.HobbyStreak, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, current_streak, longest_streak, last_completed_at, created_at FROM hobby_streaks WHERE user_id = ? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streaks []models.HobbyStreak
	for rows.Next() {
		var st models.HobbyStreak
		if err := rows.Scan(&st.ID, &st.UserID, &st.Name, &st.CurrentStreak, &st.LongestStreak, &st.LastCompletedAt, &st.CreatedAt); err != nil {
			return nil, err
		}
		streaks = append(streaks, st)
	}
	return streaks, nil
}

func (s *HobbyService) CompleteStreak(userID, streakID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var streak models.HobbyStreak
	err = tx.QueryRow(
		`SELECT id, user_id, name, current_streak, longest_streak, last_completed_at, created_at FROM hobby_streaks WHERE id = ? AND user_id = ?`,
		streakID, userID,
	).Scan(&streak.ID, &streak.UserID, &streak.Name, &streak.CurrentStreak, &streak.LongestStreak, &streak.LastCompletedAt, &streak.CreatedAt)
	if err != nil {
		return err
	}

	now := time.Now()
	streak.CurrentStreak++
	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}
	streak.LastCompletedAt = &now

	_, err = tx.Exec(
		`UPDATE hobby_streaks SET current_streak = ?, longest_streak = ?, last_completed_at = ? WHERE id = ?`,
		streak.CurrentStreak, streak.LongestStreak, streak.LastCompletedAt, streakID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO hobby_sessions (user_id, hobby_streak_id, completed_at) VALUES (?, ?, ?)`,
		userID, streakID, now,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *HobbyService) AddSession(userID int64, session *models.HobbySession) error {
	session.UserID = userID
	session.CompletedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO hobby_sessions (user_id, hobby_streak_id, duration_minutes, metric_value, metric_type, notes, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.UserID, session.HobbyStreakID, session.DurationMinutes, session.MetricValue, session.MetricType, session.Notes, session.CompletedAt,
	)
	return err
}

func (s *HobbyService) GetSessions(userID int64, limit int) ([]models.HobbySession, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, hobby_streak_id, duration_minutes, metric_value, metric_type, notes, completed_at FROM hobby_sessions WHERE user_id = ? ORDER BY completed_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.HobbySession
	for rows.Next() {
		var sess models.HobbySession
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.HobbyStreakID, &sess.DurationMinutes, &sess.MetricValue, &sess.MetricType, &sess.Notes, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

type F1Service struct {
	db *sql.DB
}

func NewF1Service(db *sql.DB) *F1Service {
	return &F1Service{db: db}
}

func (s *F1Service) CreateEvent(userID int64, event *models.F1Event) error {
	event.UserID = userID
	event.CreatedAt = time.Now()
	result, err := s.db.Exec(
		`INSERT INTO f1_events (user_id, name, event_date, event_type, created_at) VALUES (?, ?, ?, ?, ?)`,
		event.UserID, event.Name, event.EventDate, event.EventType, event.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	event.ID = id
	return nil
}

func (s *F1Service) GetUpcoming(userID int64, limit int) ([]models.F1Event, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, event_date, event_type, created_at FROM f1_events WHERE user_id = ? AND date(event_date) >= date('now') ORDER BY event_date LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.F1Event
	for rows.Next() {
		var ev models.F1Event
		if err := rows.Scan(&ev.ID, &ev.UserID, &ev.Name, &ev.EventDate, &ev.EventType, &ev.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (s *F1Service) GetAll(userID int64) ([]models.F1Event, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, event_date, event_type, created_at FROM f1_events WHERE user_id = ? ORDER BY event_date`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.F1Event
	for rows.Next() {
		var ev models.F1Event
		if err := rows.Scan(&ev.ID, &ev.UserID, &ev.Name, &ev.EventDate, &ev.EventType, &ev.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (s *F1Service) Delete(userID, eventID int64) error {
	_, err := s.db.Exec(`DELETE FROM f1_events WHERE id = ? AND user_id = ?`, eventID, userID)
	return err
}