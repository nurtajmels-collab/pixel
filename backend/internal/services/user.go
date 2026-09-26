package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetOrCreateByTelegramID(telegramID int64, username string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		`SELECT id, telegram_id, username, created_at FROM users WHERE telegram_id = ?`,
		telegramID,
	).Scan(&user.ID, &user.TelegramID, &user.Username, &user.CreatedAt)

	if err == sql.ErrNoRows {
		result, err := s.db.Exec(
			`INSERT INTO users (telegram_id, username, created_at) VALUES (?, ?, ?)`,
			telegramID, username, time.Now(),
		)
		if err != nil {
			return nil, err
		}
		id, _ := result.LastInsertId()
		user.ID = id
		user.TelegramID = telegramID
		user.Username = username
		user.CreatedAt = time.Now()
		return &user, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetByID(id int64) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		`SELECT id, telegram_id, username, created_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.TelegramID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}