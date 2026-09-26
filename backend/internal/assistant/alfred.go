package assistant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const alfredSystemPrompt = `Ты — Альфред Пенниуорт, дворецкий и строгий наставник. Твой подопечный — Мелис, ученик НИШ, готовится к SAT (3 октября 2026) и IELTS, планирует поступать на инженерию/ИИ. Он программирует (Go/React), тренируется дома, играет на гитаре, учится собирать кубик Рубика (CFOP). Твой стиль: безупречно вежливый, с британской иронией, без заискивания. Если он перерабатывает — предупреди о выгорании. Если ленится — напомни о дедлайнах. Твоя задача: вычленить активности за день и новые напоминания, вернув результат СТРОГО в JSON.`

type AlfredResponse struct {
	AlfredReply       string            `json:"alfred_reply"`
	TrackedActivities []TrackedActivity `json:"tracked_activities"`
	NewReminders      []NewReminder     `json:"new_reminders"`
}

type TrackedActivity struct {
	Category      string   `json:"category"`
	Description   string   `json:"description"`
	StartTime     *string  `json:"start_time"`
	EndTime       *string  `json:"end_time"`
	DurationHours *float64 `json:"duration_hours"`
	Tags          []string `json:"tags"`
}

type NewReminder struct {
	Task     string    `json:"task"`
	RemindAt time.Time `json:"remind_at"`
}

type AlfredService struct {
	db    *sql.DB
	key   string
	model string
	http  *http.Client
}

func NewAlfredService(db *sql.DB, key, model string) *AlfredService {
	if model == "" {
		model = "gemini-flash-latest"
	}
	return &AlfredService{db: db, key: key, model: model, http: &http.Client{Timeout: 35 * time.Second}}
}

func (a *AlfredService) AskAlfred(ctx context.Context, userText string) (*AlfredResponse, error) {
	if strings.TrimSpace(userText) == "" {
		return nil, fmt.Errorf("user text is empty")
	}
	if a.key == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}

	payload := map[string]any{
		"system_instruction": map[string]any{"parts": []map[string]string{{"text": alfredSystemPrompt}}},
		"contents":           []map[string]any{{"role": "user", "parts": []map[string]string{{"text": userText}}}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"alfred_reply": map[string]string{"type": "STRING"},
					"tracked_activities": map[string]any{"type": "ARRAY", "items": map[string]any{
						"type": "OBJECT", "properties": map[string]any{
							"category": map[string]string{"type": "STRING"}, "description": map[string]string{"type": "STRING"},
							"start_time": map[string]any{"type": "STRING", "nullable": true}, "end_time": map[string]any{"type": "STRING", "nullable": true},
							"duration_hours": map[string]any{"type": "NUMBER", "nullable": true}, "tags": map[string]any{"type": "ARRAY", "items": map[string]string{"type": "STRING"}},
						}, "required": []string{"category", "description", "tags"},
					}},
					"new_reminders": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "OBJECT", "properties": map[string]any{
						"task": map[string]string{"type": "STRING"}, "remind_at": map[string]string{"type": "STRING", "format": "DATE_TIME"},
					}, "required": []string{"task", "remind_at"}}},
				}, "required": []string{"alfred_reply", "tracked_activities", "new_reminders"},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal Gemini request: %w", err)
	}
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(a.model) + ":generateContent?key=" + url.QueryEscape(a.key)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Gemini request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	var response *http.Response
	for attempt, delay := range []time.Duration{0, time.Second, 3 * time.Second} {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		if attempt > 0 {
			requestBody, bodyErr := request.GetBody()
			if bodyErr != nil {
				return nil, fmt.Errorf("rewind Gemini request: %w", bodyErr)
			}
			request.Body = requestBody
		}
		response, err = a.http.Do(request)
		if err != nil {
			return nil, fmt.Errorf("call Gemini: %w", err)
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			break
		}
		response.Body.Close()
		if (response.StatusCode != http.StatusTooManyRequests && response.StatusCode != http.StatusInternalServerError && response.StatusCode != http.StatusBadGateway && response.StatusCode != http.StatusServiceUnavailable) || attempt == 2 {
			return nil, fmt.Errorf("Gemini returned HTTP %d", response.StatusCode)
		}
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
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Gemini envelope: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini returned no content")
	}
	var parsed AlfredResponse
	if err := json.Unmarshal([]byte(result.Candidates[0].Content.Parts[0].Text), &parsed); err != nil {
		return nil, fmt.Errorf("decode Alfred JSON: %w", err)
	}
	return &parsed, nil
}

func (a *AlfredService) StoreResponse(ctx context.Context, userID int64, response *AlfredResponse) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, activity := range response.TrackedActivities {
		tags, err := json.Marshal(activity.Tags)
		if err != nil {
			return fmt.Errorf("encode activity tags: %w", err)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO activities
			(user_id, category, description, start_time, end_time, duration_hours, tags)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, activity.Category, activity.Description,
			activity.StartTime, activity.EndTime, activity.DurationHours, string(tags))
		if err != nil {
			return fmt.Errorf("save activity: %w", err)
		}
	}
	for _, reminder := range response.NewReminders {
		if reminder.Task == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reminders (user_id, task, remind_at) VALUES (?, ?, ?)`, userID, reminder.Task, reminder.RemindAt.UTC())
		if err != nil {
			return fmt.Errorf("save reminder: %w", err)
		}
	}
	return tx.Commit()
}
