package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"pixellife-tracker/internal/assistant"
	"pixellife-tracker/internal/models"
	"pixellife-tracker/internal/services"
)

type Client struct {
	token   string
	baseURL string
	users   *services.UserService
	sleep   *services.SleepService
	sport   *services.SportService
	media   *services.MediaService
	db      *sql.DB
	alfred  *assistant.AlfredService
	http    *http.Client
}

type updateResponse struct {
	OK     bool `json:"ok"`
	Result []struct {
		UpdateID int64 `json:"update_id"`
		Message  *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
			} `json:"from"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"result"`
}

func (c *Client) Validate(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("telegram token is empty")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/getMe", nil)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram getMe returned HTTP %d", response.StatusCode)
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram getMe returned ok=false")
	}
	return nil
}

func New(token string, users *services.UserService, sleep *services.SleepService, sport *services.SportService, media *services.MediaService, database *sql.DB, alfred *assistant.AlfredService) *Client {
	return &Client{token: token, baseURL: "https://api.telegram.org/bot" + token, users: users, sleep: sleep, sport: sport, media: media, db: database, alfred: alfred, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Start(ctx context.Context) {
	if c.token == "" {
		return
	}
	offset := int64(0)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		updates, err := c.getUpdates(ctx, offset)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message != nil {
				go c.handle(update.Message.Chat.ID, update.Message.From.ID, update.Message.From.Username, update.Message.Text)
			}
		}
	}
}

func (c *Client) getUpdates(ctx context.Context, offset int64) ([]struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Text string `json:"text"`
	} `json:"message"`
}, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/getUpdates?timeout=25&offset="+strconv.FormatInt(offset, 10), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("telegram getUpdates returned HTTP %d", response.StatusCode)
	}
	var result updateResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func (c *Client) handle(chatID, telegramID int64, username, text string) {
	user, err := c.users.GetOrCreateByTelegramID(telegramID, username)
	if err != nil {
		return
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	if !strings.HasPrefix(strings.TrimSpace(text), "/") && c.alfred != nil {
		if err := c.alfred.ConfirmLatestProposal(context.Background(), user.ID, text); err == nil {
			c.send(chatID, "Подтвержденные пункты добавлены в систему, сэр.")
			_ = c.alfred.RecordConversation(context.Background(), user.ID, text, "Подтвержденные пункты сохранены")
			return
		}
		response, err := c.alfred.AskAlfredForUser(context.Background(), user.ID, text)
		if err != nil {
			c.send(chatID, "Сэр, не могу связаться с аналитическим отделом: "+err.Error())
			return
		}
		if _, err := c.alfred.CreateProposal(context.Background(), user.ID, response); err != nil {
			c.send(chatID, "Сэр, ответ подготовлен, но сохранить записи не удалось.")
			return
		}
		_ = c.alfred.RecordConversation(context.Background(), user.ID, text, response.AlfredReply)
		c.send(chatID, response.AlfredReply+"\n\n"+assistant.FormatProposal(response))
		return
	}
	switch parts[0] {
	case "/start":
		c.send(chatID, "PIXELLIFE ONLINE\n\nКоманды:\n/sleep 7.5\n/sport\n/shelf название")
	case "/sleep":
		if len(parts) != 2 {
			c.send(chatID, "Формат: /sleep 7.5")
			return
		}
		hours, parseErr := strconv.ParseFloat(parts[1], 64)
		if parseErr != nil || hours < 0 || hours > 24 {
			c.send(chatID, "Нужно число часов от 0 до 24")
			return
		}
		if c.sleep.CreateRecord(user.ID, &models.SleepRecord{SleepHours: hours}) == nil {
			c.send(chatID, fmt.Sprintf("Сон записан: %.1f ч", hours))
		}
	case "/sport":
		if c.sport.CreateSession(user.ID, &models.SportSession{WorkoutType: "Домашняя тренировка", Completed: true}) == nil {
			c.send(chatID, "Тренировка отмечена. +1 к ритму")
		}
	case "/shelf":
		if len(parts) < 2 {
			c.send(chatID, "Формат: /shelf название")
			return
		}
		title := strings.TrimSpace(strings.TrimPrefix(text, parts[0]))
		if c.media.Create(user.ID, &models.MediaItem{Type: models.MediaTypeSeries, Title: title, Status: models.MediaStatusPlanned}) == nil {
			c.send(chatID, "Добавил в полку: "+title)
		}
	default:
		c.send(chatID, "Не понял команду. Напиши /start")
	}
}

func (c *Client) SendMessage(chatID int64, text string) error { return c.send(chatID, text) }

func (c *Client) send(chatID int64, text string) error {
	_, err := c.http.PostForm(c.baseURL+"/sendMessage", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "text": {text}})
	return err
}

func (c *Client) StartReminderWorker(ctx context.Context) {
	if c.db == nil || c.token == "" {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	c.processReminders(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.processReminders(ctx)
		}
	}
}

func (c *Client) processReminders(ctx context.Context) {
	rows, err := c.db.QueryContext(ctx, `SELECT reminders.id, reminders.task, users.telegram_id
		FROM reminders JOIN users ON users.id = reminders.user_id
		WHERE reminders.is_sent = 0 AND datetime(reminders.remind_at) <= CURRENT_TIMESTAMP`)
	if err != nil {
		return
	}
	type pendingReminder struct {
		id         int64
		task       string
		telegramID int64
	}
	var pending []pendingReminder
	for rows.Next() {
		var reminder pendingReminder
		if rows.Scan(&reminder.id, &reminder.task, &reminder.telegramID) != nil {
			continue
		}
		pending = append(pending, reminder)
	}
	rows.Close()
	for _, reminder := range pending {
		if err := c.SendMessage(reminder.telegramID, "Сэр, напоминаю: "+reminder.task); err != nil {
			continue
		}
		_, _ = c.db.ExecContext(ctx, `UPDATE reminders SET is_sent = 1 WHERE id = ? AND is_sent = 0`, reminder.id)
	}
}
