package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func Init(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE,
			username TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS study_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			quality_rating INTEGER,
			notes TEXT,
			completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS study_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			score REAL NOT NULL,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS study_subjects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			fa_weight REAL NOT NULL DEFAULT 0.2,
			sau_weight REAL NOT NULL DEFAULT 0.4,
			sat_weight REAL NOT NULL DEFAULT 0.4,
			UNIQUE(user_id, name),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS study_assessments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			subject TEXT NOT NULL,
			assessment_type TEXT NOT NULL CHECK(assessment_type IN ('FA', 'SAU', 'SAT')),
			score REAL NOT NULL,
			max_score REAL NOT NULL,
			notes TEXT,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS media_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			cover_url TEXT,
			status TEXT NOT NULL DEFAULT 'planned',
			current_season INTEGER,
			current_episode INTEGER,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS sport_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			workout_type TEXT,
			completed BOOLEAN NOT NULL DEFAULT 1,
			notes TEXT,
			completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS sleep_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			sleep_hours REAL NOT NULL,
			bedtime TEXT,
			wake_time TEXT,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS hobby_streaks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			current_streak INTEGER NOT NULL DEFAULT 0,
			longest_streak INTEGER NOT NULL DEFAULT 0,
			last_completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS hobby_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			hobby_streak_id INTEGER,
			duration_minutes INTEGER,
			metric_value REAL,
			metric_type TEXT,
			notes TEXT,
			completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (hobby_streak_id) REFERENCES hobby_streaks(id)
		);`,

		`CREATE TABLE IF NOT EXISTS f1_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			event_date DATE NOT NULL,
			event_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS radar_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			study_score REAL DEFAULT 0,
			sport_score REAL DEFAULT 0,
			hobby_score REAL DEFAULT 0,
			routine_score REAL DEFAULT 0,
			calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS activities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			description TEXT NOT NULL,
			start_time TEXT,
			end_time TEXT,
			duration_hours REAL,
			tags TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			task TEXT NOT NULL,
			remind_at DATETIME NOT NULL,
			is_sent BOOLEAN NOT NULL DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS conversation_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
			text TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS assistant_proposals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected', 'expired')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_study_sessions_user ON study_sessions(user_id, completed_at);`,
		`CREATE INDEX IF NOT EXISTS idx_study_assessments_user ON study_assessments(user_id, subject, assessment_type, recorded_at);`,
		`CREATE INDEX IF NOT EXISTS idx_media_items_user ON media_items(user_id, type);`,
		`CREATE INDEX IF NOT EXISTS idx_sport_sessions_user ON sport_sessions(user_id, completed_at);`,
		`CREATE INDEX IF NOT EXISTS idx_sleep_records_user ON sleep_records(user_id, recorded_at);`,
		`CREATE INDEX IF NOT EXISTS idx_hobby_sessions_user ON hobby_sessions(user_id, completed_at);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_user ON activities(user_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_reminders_pending ON reminders(is_sent, remind_at);`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_messages_user ON conversation_messages(user_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_assistant_proposals_pending ON assistant_proposals(user_id, status, created_at);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	for _, q := range []string{
		`ALTER TABLE study_scores ADD COLUMN subject TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE study_scores ADD COLUMN component TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE study_scores ADD COLUMN max_score REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE sport_sessions ADD COLUMN duration_minutes INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sport_sessions ADD COLUMN metric_value REAL`,
		`ALTER TABLE sport_sessions ADD COLUMN metric_type TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := db.Exec(q); err != nil && !isDuplicateColumnError(err) {
			return err
		}
	}

	return nil
}

func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
