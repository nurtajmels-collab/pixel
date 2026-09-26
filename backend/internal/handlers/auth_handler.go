package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/config"
	"pixellife-tracker/internal/services"
)

type AuthHandler struct {
	userSvc *services.UserService
	cfg     *config.Config
}

func NewAuthHandler(userSvc *services.UserService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{userSvc: userSvc, cfg: cfg}
}

func (h *AuthHandler) TelegramAuth(c *fiber.Ctx) error {
	var req struct {
		TelegramID int64  `json:"telegram_id"`
		Username   string `json:"username"`
		InitData   string `json:"init_data"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.TelegramID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "telegram_id required"})
	}
	if h.cfg.AppEnv == "production" && !h.cfg.PersonalAuth {
		if !validateTelegramInitData(req.InitData, h.cfg.TelegramToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid telegram init data"})
		}
	}

	user, err := h.userSvc.GetOrCreateByTelegramID(req.TelegramID, req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	token, err := GenerateToken(user.ID, h.cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user":  user,
	})
}

func validateTelegramInitData(raw, botToken string) bool {
	if raw == "" || botToken == "" {
		return false
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return false
	}
	hash := values.Get("hash")
	values.Del("hash")
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, key+"="+strings.Join(value, ","))
	}
	sort.Strings(parts)
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	check := hmac.New(sha256.New, secret.Sum(nil))
	check.Write([]byte(strings.Join(parts, "\n")))
	if !hmac.Equal([]byte(hash), []byte(hex.EncodeToString(check.Sum(nil)))) {
		return false
	}
	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || time.Since(time.Unix(authDate, 0)) > 24*time.Hour || time.Unix(authDate, 0).After(time.Now().Add(2*time.Minute)) {
		return false
	}
	return true
}
