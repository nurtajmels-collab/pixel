package assistant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrNoPendingProposal = errors.New("no pending proposal")

const alfredSystemPrompt = `Ты — Альфред Пенниуорт, безупречно вежливый дворецкий и строгий наставник Мэлса.

Твоя задача — помогать Мэлсу анализировать ежедневную жизнь, но никогда не выдумывать его интересы, привычки или цели. Узнавай, что ему нравится, чем он занимается и что ему регулярно не удаётся выполнять, используя текущее сообщение, историю диалога, данные раздела Rhythm и подтверждённые пользователем цели.

Не считай Go, React, гитару, кубик Рубика, домашние тренировки или другие занятия интересами Мэлса, пока он сам явно о них не сообщил.

Известное расписание:
- Samsung Campus AI/ML: суббота 15:00–17:00; воскресенье 14:00–16:00.
- Школа: понедельник и среда 08:00–15:30; вторник 08:00–17:00; четверг и пятница 08:00–13:30.

Расписание является планом, а не фактически выполненной активностью. Не записывай его как выполненную учёбу, пока пользователь не сообщил, что действительно был на занятии.

Правила:
- Если пользователь сообщает о фактически выполненной активности, предложи сохранить её.
- Если пользователь сообщает о сне, предложи добавить запись сна.
- Если пользователь сообщает о регулярной цели или привычке, предложи создать streak.
- Если пользователь сообщает о спорте, предложи сохранить тренировку.
- Если пользователь говорит о книге, сериале, песне, курсе или другом media, предложи добавить это на полку.
- Если пользователь просит напоминание, предложи создать reminder.
- Не записывай ничего в систему сразу. Сначала сформируй список предложений и попроси подтверждение.
- Пользователь может подтвердить все предложения, отдельные пункты по номерам или отказаться.
- Если пользователь отвечает «да», «давай», «подтверждаю» или аналогично, подтверди предложения из последнего списка.
- Если ответ неоднозначный, задай уточняющий вопрос.
- Не говори «я записал» или «я добавил», пока пользователь не подтвердил предложения и приложение не сохранило их.
- В конце анализа показывай конкретный список предложений для streaks, sleep, sport, study, media и reminders.
- Учитывай перегрузку и риск выгорания.
- Не повторяй общий список SAT, школы и программирования без связи с текущим сообщением.

Стиль: британская вежливость, лёгкая ирония, уважение, без заискивания и чрезмерно длинных ответов.

Строгое правило синхронизации JSON:
- Массивы и объекты JSON являются единственным источником предложений для приложения.
- Никогда не описывай в alfred_reply сон, streak, активность или reminder как предложение, если соответствующий объект отсутствует в JSON.
- Если в сообщении есть факт сна, заполни sleep_record даже если время отбоя или подъёма неизвестно: укажи sleep_hours, а неизвестные bedtime и wake_time оставь пустыми строками.
- Если пользователь сообщает о регулярной цели или хочет начать занятие, добавь каждую отдельную цель в new_hobby_streaks. Не прячь цели только в alfred_reply.
- Если ты показываешь в alfred_reply нумерованный пункт для подтверждения, этот пункт обязательно должен существовать в tracked_activities, sleep_record, new_hobby_streaks или new_reminders.
- Сначала сформируй структурированные поля, затем напиши alfred_reply только на их основе.

Отвечай строго в JSON согласно предоставленной схеме.`

type AlfredResponse struct {
	AlfredReply       string            `json:"alfred_reply"`
	TrackedActivities []TrackedActivity `json:"tracked_activities"`
	NewReminders      []NewReminder     `json:"new_reminders"`
	SleepRecord       *SleepRecordInput `json:"sleep_record"`
	NewHobbyStreaks   []NewHobbyStreak  `json:"new_hobby_streaks"`
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
	Task     string `json:"task"`
	RemindAt string `json:"remind_at"`
}

type SleepRecordInput struct {
	SleepHours float64 `json:"sleep_hours"`
	Bedtime    string  `json:"bedtime"`
	WakeTime   string  `json:"wake_time"`
}

type NewHobbyStreak struct {
	Name string `json:"name"`
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
	return a.ask(ctx, userText)
}

func (a *AlfredService) AskAlfredForUser(ctx context.Context, userID int64, userText string) (*AlfredResponse, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT role, text FROM conversation_messages WHERE user_id = ? ORDER BY id DESC LIMIT 8`, userID)
	if err != nil {
		return nil, fmt.Errorf("load conversation: %w", err)
	}
	defer rows.Close()
	var messages []string
	for rows.Next() {
		var role, text string
		if err := rows.Scan(&role, &text); err != nil {
			return nil, fmt.Errorf("read conversation: %w", err)
		}
		messages = append([]string{role + ": " + text}, messages...)
	}
	if len(messages) == 0 {
		return a.ask(ctx, userText)
	}
	contextText := "Контекст предыдущего диалога (используй его для понимания подтверждений и местоимений):\n" + strings.Join(messages, "\n") + "\n\nНовое сообщение пользователя:\n" + userText
	return a.ask(ctx, contextText)
}

func (a *AlfredService) ask(ctx context.Context, userText string) (*AlfredResponse, error) {
	if strings.TrimSpace(userText) == "" {
		return nil, fmt.Errorf("user text is empty")
	}
	if a.key == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}

	payload := map[string]any{
		"system_instruction": map[string]any{"parts": []map[string]string{{"text": alfredSystemPrompt + `

Правила структурирования:
- Если пользователь сообщает, сколько спал, заполни sleep_record. Если сна нет в сообщении, верни null.
- Если пользователь хочет начать или закрепить хобби/цель, добавь ее в new_hobby_streaks.
- Фразы подтверждения вроде «давай», «зафиксируй», «запиши» относятся к цели из текущего сообщения; не утверждай, что данные сохранены, если массивы пусты.
- В tracked_activities записывай только фактически выполненные действия, а не планы.
- В alfred_reply упоминай конкретно, что было записано, и давай один следующий шаг. Не повторяй общие советы про SAT и программирование без связи с сообщением.`}}},
		"contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": userText}}}},
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
					"sleep_record": map[string]any{"type": "OBJECT", "nullable": true, "properties": map[string]any{
						"sleep_hours": map[string]string{"type": "NUMBER"}, "bedtime": map[string]string{"type": "STRING"}, "wake_time": map[string]string{"type": "STRING"},
					}, "required": []string{"sleep_hours", "bedtime", "wake_time"}},
					"new_hobby_streaks": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "OBJECT", "properties": map[string]any{
						"name": map[string]string{"type": "STRING"},
					}, "required": []string{"name"}}},
				}, "required": []string{"alfred_reply", "tracked_activities", "new_reminders", "sleep_record", "new_hobby_streaks"},
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

func (a *AlfredService) RecordConversation(ctx context.Context, userID int64, userText, assistantText string) error {
	_, err := a.db.ExecContext(ctx, `INSERT INTO conversation_messages (user_id, role, text) VALUES (?, 'user', ?), (?, 'assistant', ?)`, userID, userText, userID, assistantText)
	return err
}

func (a *AlfredService) CreateProposal(ctx context.Context, userID int64, response *AlfredResponse) (int64, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return 0, fmt.Errorf("encode proposal: %w", err)
	}
	result, err := a.db.ExecContext(ctx, `INSERT INTO assistant_proposals (user_id, payload) VALUES (?, ?)`, userID, string(payload))
	if err != nil {
		return 0, fmt.Errorf("save proposal: %w", err)
	}
	return result.LastInsertId()
}

func (a *AlfredService) ConfirmLatestProposal(ctx context.Context, userID int64, selection string) error {
	var proposalID int64
	var payload string
	err := a.db.QueryRowContext(ctx, `SELECT id, payload FROM assistant_proposals WHERE user_id = ? AND status = 'pending' ORDER BY id DESC LIMIT 1`, userID).Scan(&proposalID, &payload)
	if err == sql.ErrNoRows {
		return ErrNoPendingProposal
	}
	if err != nil {
		return err
	}
	var original AlfredResponse
	if err := json.Unmarshal([]byte(payload), &original); err != nil {
		return fmt.Errorf("decode proposal: %w", err)
	}
	confirmed, err := selectProposalItems(&original, selection)
	if err != nil {
		return err
	}
	if err := a.StoreResponse(ctx, userID, confirmed); err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE assistant_proposals SET status = 'approved' WHERE id = ?`, proposalID)
	return err
}

func selectProposalItems(original *AlfredResponse, selection string) (*AlfredResponse, error) {
	selection = strings.TrimSpace(strings.ToLower(selection))
	selectAll := selection == "всё" || selection == "все" || selection == "да" || selection == "давай" || selection == "подтверждаю" || selection == "all"
	selected := map[int]bool{}
	if selectAll {
		for index := 1; index <= proposalItemCount(original); index++ {
			selected[index] = true
		}
	} else {
		for _, part := range strings.FieldsFunc(selection, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			index, err := strconv.Atoi(part)
			if err != nil || index < 1 || index > proposalItemCount(original) {
				return nil, fmt.Errorf("укажите номера предложений, например: 1, 3")
			}
			selected[index] = true
		}
	}
	confirmed := &AlfredResponse{AlfredReply: original.AlfredReply}
	item := 0
	for _, activity := range original.TrackedActivities {
		item++
		if selected[item] {
			confirmed.TrackedActivities = append(confirmed.TrackedActivities, activity)
		}
	}
	for _, hobby := range original.NewHobbyStreaks {
		item++
		if selected[item] {
			confirmed.NewHobbyStreaks = append(confirmed.NewHobbyStreaks, hobby)
		}
	}
	if original.SleepRecord != nil {
		item++
		if selected[item] {
			confirmed.SleepRecord = original.SleepRecord
		}
	}
	for _, reminder := range original.NewReminders {
		item++
		if selected[item] {
			confirmed.NewReminders = append(confirmed.NewReminders, reminder)
		}
	}
	return confirmed, nil
}

func proposalItemCount(response *AlfredResponse) int {
	return len(response.TrackedActivities) + len(response.NewHobbyStreaks) + boolToInt(response.SleepRecord != nil) + len(response.NewReminders)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func FormatProposal(response *AlfredResponse) string {
	lines := []string{"Предлагаю добавить. Выберите номера или напишите «всё»:"}
	item := 0
	for _, activity := range response.TrackedActivities {
		item++
		lines = append(lines, fmt.Sprintf("%d. %s: %s", item, activity.Category, activity.Description))
	}
	for _, hobby := range response.NewHobbyStreaks {
		item++
		lines = append(lines, fmt.Sprintf("%d. Стрик: %s", item, hobby.Name))
	}
	if response.SleepRecord != nil {
		item++
		lines = append(lines, fmt.Sprintf("%d. Сон: %.1f ч", item, response.SleepRecord.SleepHours))
	}
	for _, reminder := range response.NewReminders {
		item++
		lines = append(lines, fmt.Sprintf("%d. Напоминание: %s (%s)", item, reminder.Task, reminder.RemindAt))
	}
	return strings.Join(lines, "\n")
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
		remindAt, err := parseReminderTime(reminder.RemindAt)
		if err != nil {
			return fmt.Errorf("parse reminder time %q: %w", reminder.RemindAt, err)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reminders (user_id, task, remind_at) VALUES (?, ?, ?)`, userID, reminder.Task, remindAt.UTC())
		if err != nil {
			return fmt.Errorf("save reminder: %w", err)
		}
	}
	if response.SleepRecord != nil && response.SleepRecord.SleepHours > 0 && response.SleepRecord.SleepHours <= 24 {
		_, err = tx.ExecContext(ctx, `INSERT INTO sleep_records (user_id, sleep_hours, bedtime, wake_time) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''))`, userID, response.SleepRecord.SleepHours, response.SleepRecord.Bedtime, response.SleepRecord.WakeTime)
		if err != nil {
			return fmt.Errorf("save sleep record: %w", err)
		}
	}
	for _, hobby := range response.NewHobbyStreaks {
		if strings.TrimSpace(hobby.Name) == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO hobby_streaks (user_id, name) VALUES (?, ?)`, userID, strings.TrimSpace(hobby.Name))
		if err != nil {
			return fmt.Errorf("save hobby streak: %w", err)
		}
	}
	return tx.Commit()
}

func parseReminderTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported datetime format")
}
