package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type StudyService struct {
	db *sql.DB
}

func NewStudyService(db *sql.DB) *StudyService {
	return &StudyService{db: db}
}

func (s *StudyService) CreateSession(userID int64, session *models.StudySession) error {
	session.UserID = userID
	session.CompletedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO study_sessions (user_id, category, duration_minutes, quality_rating, notes, completed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		session.UserID, session.Category, session.DurationMinutes, session.QualityRating, session.Notes, session.CompletedAt,
	)
	return err
}

func (s *StudyService) GetSessions(userID int64, limit int) ([]models.StudySession, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, category, duration_minutes, quality_rating, notes, completed_at FROM study_sessions WHERE user_id = ? ORDER BY completed_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.StudySession
	for rows.Next() {
		var sess models.StudySession
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.Category, &sess.DurationMinutes, &sess.QualityRating, &sess.Notes, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *StudyService) GetTodayMinutes(userID int64) (int, error) {
	var total int
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(duration_minutes), 0) FROM study_sessions WHERE user_id = ? AND date(completed_at) = date('now')`,
		userID,
	).Scan(&total)
	return total, err
}

func (s *StudyService) AddScore(userID int64, category, subject, component string, score, maxScore float64) error {
	_, err := s.db.Exec(
		`INSERT INTO study_scores (user_id, category, subject, component, score, max_score, recorded_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, category, subject, component, score, maxScore, time.Now(),
	)
	return err
}

func (s *StudyService) GetScores(userID int64, category string) ([]models.StudyScore, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, category, subject, component, score, max_score, recorded_at FROM study_scores WHERE user_id = ? AND category = ? ORDER BY recorded_at`,
		userID, category,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []models.StudyScore
	for rows.Next() {
		var sc models.StudyScore
		if err := rows.Scan(&sc.ID, &sc.UserID, &sc.Category, &sc.Subject, &sc.Component, &sc.Score, &sc.MaxScore, &sc.RecordedAt); err != nil {
			return nil, err
		}
		scores = append(scores, sc)
	}
	return scores, nil
}

func (s *StudyService) DeleteScore(userID, scoreID int64) error {
	_, err := s.db.Exec(`DELETE FROM study_scores WHERE id = ? AND user_id = ?`, scoreID, userID)
	return err
}

func (s *StudyService) AddAssessment(userID int64, assessment *models.StudyAssessment) error {
	assessment.UserID = userID
	assessment.RecordedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO study_assessments (user_id, subject, assessment_type, score, max_score, notes, recorded_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, assessment.UserID, assessment.Subject, assessment.AssessmentType, assessment.Score, assessment.MaxScore, assessment.Notes, assessment.RecordedAt)
	return err
}

func (s *StudyService) GetAssessments(userID int64, subject string) ([]models.StudyAssessment, error) {
	rows, err := s.db.Query(`SELECT id, user_id, subject, assessment_type, score, max_score, notes, recorded_at FROM study_assessments WHERE user_id = ? AND (? = '' OR subject = ?) ORDER BY recorded_at DESC`, userID, subject, subject)
	if err != nil { return nil, err }
	defer rows.Close()
	assessments := []models.StudyAssessment{}
	for rows.Next() {
		var assessment models.StudyAssessment
		if err := rows.Scan(&assessment.ID, &assessment.UserID, &assessment.Subject, &assessment.AssessmentType, &assessment.Score, &assessment.MaxScore, &assessment.Notes, &assessment.RecordedAt); err != nil { return nil, err }
		assessments = append(assessments, assessment)
	}
	return assessments, rows.Err()
}

func (s *StudyService) GetSummary(userID int64, subject string) (map[string]any, error) {
	var faWeight, sauWeight, satWeight float64
	err := s.db.QueryRow(`SELECT fa_weight, sau_weight, sat_weight FROM study_subjects WHERE user_id = ? AND name = ?`, userID, subject).Scan(&faWeight, &sauWeight, &satWeight)
	if err == sql.ErrNoRows { faWeight, sauWeight, satWeight = 0.2, 0.4, 0.4 }
	if err != nil && err != sql.ErrNoRows { return nil, err }
	rows, err := s.db.Query(`SELECT assessment_type, score, max_score FROM study_assessments WHERE user_id = ? AND subject = ?`, userID, subject)
	if err != nil { return nil, err }
	defer rows.Close()
	values := map[string][]float64{"FA": {}, "SAU": {}, "SAT": {}}
	for rows.Next() {
		var assessmentType string
		var score, maxScore float64
		if err := rows.Scan(&assessmentType, &score, &maxScore); err != nil { return nil, err }
		if maxScore > 0 { values[assessmentType] = append(values[assessmentType], score/maxScore*100) }
	}
	average := func(items []float64) float64 { if len(items) == 0 { return 0 }; total := 0.0; for _, item := range items { total += item }; return total / float64(len(items)) }
	fa, sau, sat := average(values["FA"]), average(values["SAU"]), average(values["SAT"])
	final := fa*faWeight + sau*sauWeight + sat*satWeight
	grade := "C"
	if final >= 85 { grade = "A" } else if final >= 65 { grade = "B" }
	return map[string]any{"subject": subject, "fa_average": fa, "sau_average": sau, "sat_average": sat, "final_percent": final, "grade": grade, "weights": map[string]float64{"FA": faWeight, "SAU": sauWeight, "SAT": satWeight}}, rows.Err()
}

func (s *StudyService) SetWeights(userID int64, subject string, fa, sau, sat float64) error {
	_, err := s.db.Exec(`INSERT INTO study_subjects (user_id, name, fa_weight, sau_weight, sat_weight) VALUES (?, ?, ?, ?, ?) ON CONFLICT(user_id, name) DO UPDATE SET fa_weight=excluded.fa_weight, sau_weight=excluded.sau_weight, sat_weight=excluded.sat_weight`, userID, subject, fa, sau, sat)
	return err
}