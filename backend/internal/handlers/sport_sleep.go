package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/models"
	"pixellife-tracker/internal/services"
)

type SportHandler struct {
	svc *services.SportService
}

func NewSportHandler(svc *services.SportService) *SportHandler {
	return &SportHandler{svc: svc}
}

func (h *SportHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/sessions", h.CreateSession)
	r.Get("/sessions", h.GetSessions)
	r.Get("/today", h.GetTodayDone)
}

func (h *SportHandler) CreateSession(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.SportSession
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.CreateSession(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *SportHandler) GetSessions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	sessions, err := h.svc.GetSessions(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sessions)
}

func (h *SportHandler) GetTodayDone(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	done, err := h.svc.GetTodayDone(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"done": done})
}

type SleepHandler struct {
	svc *services.SleepService
}

func NewSleepHandler(svc *services.SleepService) *SleepHandler {
	return &SleepHandler{svc: svc}
}

func (h *SleepHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/records", h.CreateRecord)
	r.Get("/records", h.GetRecords)
	r.Get("/today", h.GetTodayHours)
}

func (h *SleepHandler) CreateRecord(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.SleepRecord
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.CreateRecord(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *SleepHandler) GetRecords(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	records, err := h.svc.GetRecords(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(records)
}

func (h *SleepHandler) GetTodayHours(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	hours, err := h.svc.GetTodayHours(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"hours": hours})
}