package services

import (
	"testing"

	"pixellife-tracker/internal/db"
)

func TestRadarRecalculatesAIActivities(t *testing.T) {
	database, err := db.Init(t.TempDir() + "/pixellife.db")
	if err != nil {
		t.Fatalf("init database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`INSERT INTO users (telegram_id) VALUES (?)`, 987654321); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	service := NewRadarService(database)
	first, err := service.CalculateAndStore(1)
	if err != nil {
		t.Fatalf("first radar calculation: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO activities (user_id, category, description, duration_hours, tags) VALUES (?, ?, ?, ?, ?)`, 1, "study", "SAT mathematics", 3, "[]"); err != nil {
		t.Fatalf("insert AI activity: %v", err)
	}
	second, err := service.CalculateAndStore(1)
	if err != nil {
		t.Fatalf("second radar calculation: %v", err)
	}
	if second.StudyScore <= first.StudyScore {
		t.Fatalf("expected study score to increase, got %.2f then %.2f", first.StudyScore, second.StudyScore)
	}
}
