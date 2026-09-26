package assistant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type Assistant struct {
	db    *sql.DB
	key   string
	model string
	bot   MessageSender
	http  *http.Client
}

func New(db *sql.DB, key, model string, telegram MessageSender) *Assistant {
	return &Assistant{db: db, key: key, model: model, bot: telegram, http: &http.Client{Timeout: 35 * time.Second}}
}

func (a *Assistant) Start(ctx context.Context) {
	if a.key == "" || a.bot == nil {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if now.Weekday() == time.Sunday {
				go a.generate(ctx)
			}
		}
	}
}

func (a *Assistant) GenerateNow(ctx context.Context) int {
	if a.key == "" || a.bot == nil {
		return 0
	}
	sent := 0
	rows, err := a.db.QueryContext(ctx, `SELECT id, telegram_id FROM users WHERE telegram_id IS NOT NULL`)
	if err != nil {
		return 0
	}
	defer rows.Close()
	for rows.Next() {
		var userID, telegramID int64
		if rows.Scan(&userID, &telegramID) != nil {
			continue
		}
		report := a.askGemini(ctx, a.weekPrompt(ctx, userID))
		if report != "" && a.bot.SendMessage(telegramID, "PIXELLIFE WEEKLY REPORT\n\n"+report) == nil {
			sent++
		}
	}
	return sent
}

func (a *Assistant) generate(ctx context.Context) {
	var userID, telegramID int64
	rows, err := a.db.QueryContext(ctx, `SELECT id, telegram_id FROM users WHERE telegram_id IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		if rows.Scan(&userID, &telegramID) != nil {
			continue
		}
		prompt := a.weekPrompt(ctx, userID)
		if prompt == "" {
			continue
		}
		report := a.askGemini(ctx, prompt)
		if report != "" {
			_ = a.bot.SendMessage(telegramID, "WEEKLY QUEST REPORT\n\n"+report)
		}
	}
}

func (a *Assistant) weekPrompt(ctx context.Context, userID int64) string {
	var study, sleep float64
	var sport int
	_ = a.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(duration_minutes), 0) FROM study_sessions WHERE user_id = ? AND completed_at >= date('now', '-7 days')`, userID).Scan(&study)
	_ = a.db.QueryRowContext(ctx, `SELECT COALESCE(AVG(sleep_hours), 0) FROM sleep_records WHERE user_id = ? AND recorded_at >= date('now', '-7 days')`, userID).Scan(&sleep)
	_ = a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sport_sessions WHERE user_id = ? AND completed = 1 AND completed_at >= date('now', '-7 days')`, userID).Scan(&sport)
	return fmt.Sprintf("Write a short Telegram report in Russian with a pixel-game voice. Study: %.0f minutes, average sleep: %.1f hours, workouts: %d. If the week is strong, hype the user like a coach. If they procrastinated, use playful NPC-villain sarcasm without insults. Include one win, one growth area and one concrete quest for next week. Keep it under 500 characters. The report will be sent directly to Telegram.", study, sleep, sport)
}

func (a *Assistant) askGemini(ctx context.Context, prompt string) string {
	payload := map[string]any{"contents": []map[string]any{{"parts": []map[string]string{{"text": prompt}}}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://generativelanguage.googleapis.com/v1beta/models/"+a.model+":generateContent?key="+a.key, bytes.NewReader(body))
	if err != nil {
		return ""
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := a.http.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.NewDecoder(response.Body).Decode(&result) != nil || len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	return result.Candidates[0].Content.Parts[0].Text
}
