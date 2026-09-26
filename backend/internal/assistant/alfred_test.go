package assistant

import (
	"context"
	"testing"
	"time"

	"pixellife-tracker/internal/db"
)

func TestStoreResponse(t *testing.T) {
	database, err := db.Init(t.TempDir() + "/pixellife.db")
	if err != nil {
		t.Fatalf("init database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`INSERT INTO users (telegram_id, username) VALUES (?, ?)`, 12345, "alfred-test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	service := NewAlfredService(database, "", "")
	duration := 1.5
	start := "2026-09-26T09:00:00+05:00"
	remindAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	response := &AlfredResponse{
		TrackedActivities: []TrackedActivity{{Category: "study", Description: "SAT mathematics", StartTime: &start, DurationHours: &duration, Tags: []string{"sat", "math"}}},
		NewReminders:      []NewReminder{{Task: "Review mistakes", RemindAt: remindAt}},
	}

	if err := service.StoreResponse(context.Background(), 1, response); err != nil {
		t.Fatalf("store response: %v", err)
	}

	var activityCount, reminderCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM activities WHERE user_id = 1`).Scan(&activityCount); err != nil {
		t.Fatalf("count activities: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM reminders WHERE user_id = 1 AND is_sent = 0`).Scan(&reminderCount); err != nil {
		t.Fatalf("count reminders: %v", err)
	}
	if activityCount != 1 || reminderCount != 1 {
		t.Fatalf("expected one activity and reminder, got %d and %d", activityCount, reminderCount)
	}
}

func TestStoreResponseRollsBackOnFailure(t *testing.T) {
	database, err := db.Init(t.TempDir() + "/pixellife.db")
	if err != nil {
		t.Fatalf("init database: %v", err)
	}
	defer database.Close()

	service := NewAlfredService(database, "", "")
	response := &AlfredResponse{TrackedActivities: []TrackedActivity{{Category: "study", Description: "orphan", Tags: []string{}}}}
	if err := service.StoreResponse(context.Background(), 999, response); err == nil {
		t.Fatal("expected foreign key failure")
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM activities`).Scan(&count); err != nil {
		t.Fatalf("count activities: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback, found %d activities", count)
	}
}
