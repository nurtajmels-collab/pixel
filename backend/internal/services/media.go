package services

import (
	"database/sql"
	"time"

	"pixellife-tracker/internal/models"
)

type MediaService struct {
	db *sql.DB
}

func NewMediaService(db *sql.DB) *MediaService {
	return &MediaService{db: db}
}

func (s *MediaService) Create(userID int64, item *models.MediaItem) error {
	item.UserID = userID
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO media_items (user_id, type, title, cover_url, status, current_season, current_episode, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.UserID, item.Type, item.Title, item.CoverURL, item.Status, item.CurrentSeason, item.CurrentEpisode, item.Notes, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (s *MediaService) GetByUser(userID int64, mediaType string) ([]models.MediaItem, error) {
	query := `SELECT id, user_id, type, title, cover_url, status, current_season, current_episode, notes, created_at, updated_at FROM media_items WHERE user_id = ?`
	args := []interface{}{userID}
	if mediaType != "" {
		query += ` AND type = ?`
		args = append(args, mediaType)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MediaItem
	for rows.Next() {
		var item models.MediaItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Title, &item.CoverURL, &item.Status, &item.CurrentSeason, &item.CurrentEpisode, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MediaService) Update(item *models.MediaItem) error {
	item.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		`UPDATE media_items SET title=?, cover_url=?, status=?, current_season=?, current_episode=?, notes=?, updated_at=? WHERE id=? AND user_id=?`,
		item.Title, item.CoverURL, item.Status, item.CurrentSeason, item.CurrentEpisode, item.Notes, item.UpdatedAt, item.ID, item.UserID,
	)
	return err
}

func (s *MediaService) Delete(userID, itemID int64) error {
	_, err := s.db.Exec(`DELETE FROM media_items WHERE id = ? AND user_id = ?`, itemID, userID)
	return err
}

func (s *MediaService) GetRecent(userID int64, limit int) ([]models.MediaItem, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, type, title, cover_url, status, current_season, current_episode, notes, created_at, updated_at FROM media_items WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MediaItem
	for rows.Next() {
		var item models.MediaItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Title, &item.CoverURL, &item.Status, &item.CurrentSeason, &item.CurrentEpisode, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}