package assistant

import (
	"context"
	"testing"

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
	response := &AlfredResponse{
		TrackedActivities: []TrackedActivity{{Category: "study", Description: "SAT mathematics", StartTime: &start, DurationHours: &duration, Tags: []string{"sat", "math"}}},
		NewReminders:      []NewReminder{{Task: "Review mistakes", RemindAt: "2026-09-27 08:00"}},
		SleepRecord:       &SleepRecordInput{SleepHours: 8, Bedtime: "23:00", WakeTime: "07:00"},
		NewHobbyStreaks:   []NewHobbyStreak{{Name: "Learn Hozier song on guitar"}},
	}

	if err := service.StoreResponse(context.Background(), 1, response); err != nil {
		t.Fatalf("store response: %v", err)
	}

	var activityCount, reminderCount, sleepCount, hobbyCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM activities WHERE user_id = 1`).Scan(&activityCount); err != nil {
		t.Fatalf("count activities: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM reminders WHERE user_id = 1 AND is_sent = 0`).Scan(&reminderCount); err != nil {
		t.Fatalf("count reminders: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM sleep_records WHERE user_id = 1 AND sleep_hours = 8`).Scan(&sleepCount); err != nil {
		t.Fatalf("count sleep records: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM hobby_streaks WHERE user_id = 1 AND name = ?`, "Learn Hozier song on guitar").Scan(&hobbyCount); err != nil {
		t.Fatalf("count hobby streaks: %v", err)
	}
	if activityCount != 1 || reminderCount != 1 || sleepCount != 1 || hobbyCount != 1 {
		t.Fatalf("expected one activity, reminder, sleep record and hobby streak, got %d, %d, %d and %d", activityCount, reminderCount, sleepCount, hobbyCount)
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

func TestProposalRequiresConfirmation(t *testing.T) {
	database, err := db.Init(t.TempDir() + "/pixellife.db")
	if err != nil {
		t.Fatalf("init database: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec(`INSERT INTO users (telegram_id) VALUES (?)`, 54321); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	service := NewAlfredService(database, "", "")
	response := &AlfredResponse{
		SleepRecord:     &SleepRecordInput{SleepHours: 8, WakeTime: "07:00"},
		NewHobbyStreaks: []NewHobbyStreak{{Name: "Гитара"}},
	}
	if _, err := service.CreateProposal(context.Background(), 1, response); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	var before int
	if err := database.QueryRow(`SELECT COUNT(*) FROM sleep_records`).Scan(&before); err != nil {
		t.Fatalf("count before confirmation: %v", err)
	}
	if before != 0 {
		t.Fatalf("proposal wrote sleep before confirmation: %d", before)
	}
	if err := service.ConfirmLatestProposal(context.Background(), 1, "1"); err != nil {
		t.Fatalf("confirm hobby proposal: %v", err)
	}
	var sleepCount, hobbyCount int
	_ = database.QueryRow(`SELECT COUNT(*) FROM sleep_records`).Scan(&sleepCount)
	_ = database.QueryRow(`SELECT COUNT(*) FROM hobby_streaks WHERE name = 'Гитара'`).Scan(&hobbyCount)
	if sleepCount != 0 || hobbyCount != 1 {
		t.Fatalf("expected only hobby after confirmation, got sleep=%d hobby=%d", sleepCount, hobbyCount)
	}
}
